package describe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	kaytuTrace "github.com/kaytu-io/kaytu-util/pkg/trace"
	"go.opentelemetry.io/otel"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	apimeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	apiCompliance "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	apiDescribe "github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/enums"
	apiInsight "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	apiOnboard "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-util/pkg/concurrency"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/meta"
)

const (
	MaxQueued = 5000
)

var (
	ErrJobInProgress = errors.New("job already in progress")
)

type CloudNativeCall struct {
	dc  DescribeConnectionJob
	src *apiOnboard.Connection
}

func (s *Scheduler) RunDescribeJobScheduler() {
	s.logger.Info("Scheduling describe jobs on a timer")

	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleDescribeJob()
	}
}

func (s *Scheduler) RunStackScheduler() {
	s.logger.Info("Scheduling stack jobs on a timer")

	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()

	for ; ; <-t.C {
		err := s.scheduleStackJobs()
		if err != nil {
			s.logger.Error(fmt.Sprintf("Scheduling stack jobs error: %v", err.Error()))
		}
	}
}

func (s *Scheduler) RunDescribeResourceJobCycle(ctx context.Context) error {
	ctx, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx, kaytuTrace.GetCurrentFuncName())
	defer span.End()

	count, err := s.db.CountQueuedDescribeConnectionJobs()
	if err != nil {
		s.logger.Error("failed to get queue length", zap.String("spot", "CountQueuedDescribeConnectionJobs"), zap.Error(err))
		DescribeResourceJobsCount.WithLabelValues("failure").Inc()
		return err
	}

	if count > MaxQueued {
		DescribePublishingBlocked.WithLabelValues("cloud queued").Set(1)
		s.logger.Error("queue is full", zap.String("spot", "count > MaxQueued"), zap.Error(err))
		return errors.New("queue is full")
	} else {
		DescribePublishingBlocked.WithLabelValues("cloud queued").Set(0)
	}

	dcs, err := s.db.ListRandomCreatedDescribeConnectionJobs(ctx, int(s.MaxConcurrentCall))
	if err != nil {
		s.logger.Error("failed to fetch describe resource jobs", zap.String("spot", "ListRandomCreatedDescribeResourceJobs"), zap.Error(err))
		DescribeResourceJobsCount.WithLabelValues("failure").Inc()
		return err
	}
	s.logger.Info("got the jobs", zap.Int("length", len(dcs)))

	counts, err := s.db.CountRunningDescribeJobsPerResourceType()
	if err != nil {
		s.logger.Error("failed to resource type count", zap.String("spot", "CountRunningDescribeJobsPerResourceType"), zap.Error(err))
		DescribeResourceJobsCount.WithLabelValues("failure").Inc()
		return err
	}

	if len(dcs) == 0 {
		if count == 0 {
			dcs, err = s.db.GetFailedDescribeConnectionJobs(ctx)
			if err != nil {
				s.logger.Error("failed to fetch failed describe resource jobs", zap.String("spot", "GetFailedDescribeResourceJobs"), zap.Error(err))
				DescribeResourceJobsCount.WithLabelValues("failure").Inc()
				return err
			}
			if len(dcs) == 0 {
				return errors.New("no job to run")
			}
		} else {
			return errors.New("queue is not empty to look for retries")
		}
	}

	rtCount := map[string]int{}
	for i := 0; i < len(dcs); i++ {
		dc := dcs[i]
		rtCount[dc.ResourceType]++

		maxCount := 25
		if m, ok := es.ResourceRateLimit[dc.ResourceType]; ok {
			maxCount = m
		}

		currentCount := 0
		for _, c := range counts {
			if c.ResourceType == dc.ResourceType {
				currentCount = c.Count
			}
		}
		if rtCount[dc.ResourceType]+currentCount > maxCount {
			if dc.ResourceType == "Microsoft.CostManagement/CostByResourceType" {
				s.logger.Info("ignoring azure cost", zap.Int("currentCount", currentCount), zap.Int("rtCount", rtCount[dc.ResourceType]))
			}

			dcs = append(dcs[:i], dcs[i+1:]...)
			i--
		}
	}

	s.logger.Info("preparing resource jobs to run", zap.Int("length", len(dcs)))

	wp := concurrency.NewWorkPool(len(dcs))
	srcMap := map[string]*apiOnboard.Connection{}
	for _, dc := range dcs {
		var src *apiOnboard.Connection
		if v, ok := srcMap[dc.ConnectionID]; ok {
			src = v
		} else {
			switch dc.TriggerType {
			case enums.DescribeTriggerTypeStack:
			default:
				src, err = s.onboardClient.GetSource(&httpclient.Context{UserRole: apiAuth.KaytuAdminRole}, dc.ConnectionID)
				if err != nil {
					s.logger.Error("failed to get source", zap.String("spot", "GetSourceByUUID"), zap.Error(err), zap.Uint("jobID", dc.ID))
					DescribeResourceJobsCount.WithLabelValues("failure").Inc()
					return err
				}

				if src.CredentialType == apiOnboard.CredentialTypeManualAwsOrganization &&
					strings.HasPrefix(strings.ToLower(dc.ResourceType), "aws::costexplorer") {
					// cost on org
				} else {
					if !src.IsEnabled() {
						continue
					}
				}
				srcMap[dc.ConnectionID] = src
			}
		}

		switch dc.TriggerType {
		case enums.DescribeTriggerTypeStack:
			cred, err := s.db.GetStackCredential(dc.ConnectionID)
			if err != nil {
				s.logger.Error("failed to get stack credential", zap.String("spot", "GetStackCredential"), zap.Error(err), zap.Uint("jobID", dc.ID))
				return err
			}
			if cred.Secret == "" {
				s.logger.Error("failed to get stack credential secret", zap.String("spot", "GetStackCredential"), zap.Error(err), zap.Uint("jobID", dc.ID))
				return errors.New(fmt.Sprintf("No secret found for %s", dc.ConnectionID))
			}
			c := CloudNativeCall{
				dc: dc,
			}
			wp.AddJob(func() (interface{}, error) {
				err := s.enqueueCloudNativeDescribeJob(ctx, c.dc, cred.Secret, s.WorkspaceName, dc.ConnectionID)
				if err != nil {
					s.logger.Error("Failed to enqueueCloudNativeDescribeConnectionJob", zap.Error(err), zap.Uint("jobID", dc.ID))
					DescribeResourceJobsCount.WithLabelValues("failure").Inc()
					return nil, err
				}
				DescribeResourceJobsCount.WithLabelValues("successful").Inc()
				return nil, nil
			})
		default:
			c := CloudNativeCall{
				dc:  dc,
				src: src,
			}
			wp.AddJob(func() (interface{}, error) {
				err := s.enqueueCloudNativeDescribeJob(ctx, c.dc, c.src.Credential.Config.(string), s.WorkspaceName, s.kafkaResourcesTopic)
				if err != nil {
					DescribeResourceJobsCount.WithLabelValues("failure").Inc()
					return nil, err
				}
				DescribeResourceJobsCount.WithLabelValues("successful").Inc()
				return nil, nil
			})
		}
	}

	res := wp.Run()
	for _, r := range res {
		if r.Error != nil {
			s.logger.Error("failure on calling cloudNative describer", zap.Error(err))
		}
	}

	return nil
}

func (s *Scheduler) RunDescribeResourceJobs(ctx context.Context) {
	t := time.NewTicker(time.Second * 1)
	defer t.Stop()
	for ; ; <-t.C {
		if err := s.RunDescribeResourceJobCycle(ctx); err != nil {
			t.Reset(time.Second * 5)
		} else {
			t.Reset(time.Second * 1)
		}
	}
}

func (s *Scheduler) scheduleDescribeJob() {
	err := s.CheckWorkspaceResourceLimit()
	if err != nil {
		s.logger.Error("failed to get limits", zap.String("spot", "CheckWorkspaceResourceLimit"), zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	connections, err := s.onboardClient.ListSources(&httpclient.Context{UserRole: apiAuth.KaytuAdminRole}, nil)
	if err != nil {
		s.logger.Error("failed to get list of sources", zap.String("spot", "ListSources"), zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	for _, connection := range connections {
		var resourceTypes []string
		switch connection.Connector {
		case source.CloudAWS:
			resourceTypes = aws.ListResourceTypes()
		case source.CloudAzure:
			resourceTypes = azure.ListResourceTypes()
		}

		for _, resourceType := range resourceTypes {
			err = s.describe(connection, resourceType, true, false)
			if err != nil {
				s.logger.Error("failed to describe connection", zap.String("connection_id", connection.ID.String()), zap.String("resource_type", resourceType), zap.Error(err))
			}
		}
	}

	DescribeJobsCount.WithLabelValues("successful").Inc()
}

func (s *Scheduler) describe(connection apiOnboard.Connection, resourceType string, scheduled, costFullDiscovery bool) error {
	if connection.CredentialType == apiOnboard.CredentialTypeManualAwsOrganization &&
		strings.HasPrefix(strings.ToLower(resourceType), "aws::costexplorer") {
		// cost on org
	} else {
		if !connection.IsEnabled() {
			return nil
		}
	}

	job, err := s.db.GetLastDescribeConnectionJob(connection.ID.String(), resourceType)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		return err
	}

	if job != nil {
		if scheduled {
			interval := s.fullDiscoveryIntervalHours
			if connection.Connector == source.CloudAWS {
				rt, _ := aws.GetResourceType(resourceType)
				if rt != nil {
					if rt.FastDiscovery {
						interval = s.describeIntervalHours
					} else if rt.CostDiscovery {
						interval = 24
					}
				}
			} else if connection.Connector == source.CloudAzure {
				rt, _ := azure.GetResourceType(resourceType)
				if rt != nil {
					if rt.FastDiscovery {
						interval = s.describeIntervalHours
					} else if rt.CostDiscovery {
						interval = 24
					}
				}
			}

			if job.UpdatedAt.After(time.Now().Add(time.Duration(-interval) * time.Hour)) {
				return nil
			}
		}

		if job.Status == api.DescribeResourceJobCreated ||
			job.Status == api.DescribeResourceJobQueued ||
			job.Status == api.DescribeResourceJobInProgress {
			return ErrJobInProgress
		}
	}

	if connection.LastHealthCheckTime.Before(time.Now().Add(-1 * 24 * time.Hour)) {
		healthCheckedSrc, err := s.onboardClient.GetSourceHealthcheck(&httpclient.Context{
			UserRole: apiAuth.EditorRole,
		}, connection.ID.String(), false)
		if err != nil {
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			return err
		}
		connection = *healthCheckedSrc
	}

	if scheduled && connection.AssetDiscoveryMethod != source.AssetDiscoveryMethodTypeScheduled {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		return errors.New("asset discovery is not scheduled")
	}

	if connection.CredentialType == apiOnboard.CredentialTypeManualAwsOrganization &&
		strings.HasPrefix(strings.ToLower(resourceType), "aws::costexplorer") {
		// cost on org
	} else {
		if (connection.LifecycleState != apiOnboard.ConnectionLifecycleStateOnboard &&
			connection.LifecycleState != apiOnboard.ConnectionLifecycleStateInProgress) ||
			connection.HealthState != source.HealthStatusHealthy {
			//DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			//return errors.New("connection is not healthy or disabled")
			return nil
		}
	}

	triggerType := enums.DescribeTriggerTypeScheduled
	if connection.LifecycleState == apiOnboard.ConnectionLifecycleStateInProgress {
		triggerType = enums.DescribeTriggerTypeInitialDiscovery
	}
	if costFullDiscovery {
		triggerType = enums.DescribeTriggerTypeCostFullDiscovery
	}
	s.logger.Debug("Connection is due for a describe. Creating a job now", zap.String("connectionID", connection.ID.String()), zap.String("resourceType", resourceType))
	daj := newDescribeConnectionJob(connection, resourceType, triggerType)
	err = s.db.CreateDescribeConnectionJob(&daj)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		return err
	}
	DescribeSourceJobsCount.WithLabelValues("successful").Inc()

	if connection.LifecycleState == apiOnboard.ConnectionLifecycleStateInProgress {
		_, err = s.onboardClient.SetConnectionLifecycleState(&httpclient.Context{
			UserRole: apiAuth.EditorRole,
		}, connection.ID.String(), apiOnboard.ConnectionLifecycleStateOnboard)
		if err != nil {
			s.logger.Warn("Failed to set connection lifecycle state", zap.String("connection_id", connection.ID.String()), zap.Error(err))
		}
	}

	return nil
}

func newDescribeConnectionJob(a apiOnboard.Connection, resourceType string, triggerType enums.DescribeTriggerType) DescribeConnectionJob {
	return DescribeConnectionJob{
		ConnectionID: a.ID.String(),
		Connector:    a.Connector,
		AccountID:    a.ConnectionID,
		TriggerType:  triggerType,
		ResourceType: resourceType,
		Status:       apiDescribe.DescribeResourceJobCreated,
	}
}

func (s *Scheduler) enqueueCloudNativeDescribeJob(ctx context.Context, dc DescribeConnectionJob, cipherText string, workspaceName string, kafkaTopic string) error {
	ctx, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx, kaytuTrace.GetCurrentFuncName())
	defer span.End()

	s.logger.Debug("enqueueCloudNativeDescribeJob",
		zap.Uint("jobID", dc.ID),
		zap.String("connectionID", dc.ConnectionID),
		zap.String("resourceType", dc.ResourceType),
	)

	input := LambdaDescribeWorkerInput{
		WorkspaceId:      CurrentWorkspaceID,
		WorkspaceName:    workspaceName,
		DescribeEndpoint: s.describeEndpoint,
		KeyARN:           s.keyARN,
		KeyRegion:        s.keyRegion,
		KafkaTopic:       kafkaTopic,
		DescribeJob: DescribeJob{
			JobID:        dc.ID,
			ResourceType: dc.ResourceType,
			SourceID:     dc.ConnectionID,
			AccountID:    dc.AccountID,
			DescribedAt:  dc.CreatedAt.UnixMilli(),
			SourceType:   dc.Connector,
			CipherText:   cipherText,
			TriggerType:  dc.TriggerType,
			RetryCounter: 0,
		},
	}
	lambdaRequest, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal cloud native req due to %v", err)
	}

	httpClient := &http.Client{
		Timeout: 1 * time.Minute,
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/kaytu-%s-describer", LambdaFuncsBaseURL, strings.ToLower(dc.Connector.String())), bytes.NewBuffer(lambdaRequest))
	if err != nil {
		return fmt.Errorf("failed to create http request due to %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send orchestrators http request due to %v", err)
	}

	defer resp.Body.Close()
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read orchestrators http response due to %v", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("failed to trigger cloud native worker due to %d: %s", resp.StatusCode, string(resBody))
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to trigger cloud native worker due to %d: %s", resp.StatusCode, string(resBody))
	}

	s.logger.Info("successful job trigger",
		zap.Uint("jobID", dc.ID),
		zap.String("connectionID", dc.ConnectionID),
		zap.String("resourceType", dc.ResourceType),
	)

	if err := s.db.QueueDescribeConnectionJob(dc.ID); err != nil {
		s.logger.Error("failed to QueueDescribeResourceJob",
			zap.Uint("jobID", dc.ID),
			zap.String("connectionID", dc.ConnectionID),
			zap.String("resourceType", dc.ResourceType),
			zap.Error(err),
		)
	}
	return nil
}

// ================================================ STACKS ================================================

func (s *Scheduler) scheduleStackJobs() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	s.logger.Info("Schedule stack jobs started")

	kubeClient, err := s.httpServer.newKubeClient()
	if err != nil {
		s.logger.Error(fmt.Sprintf("failt to make new kube client: %s", err.Error()))
		return fmt.Errorf("failt to make new kube client: %w", err)
	}
	s.httpServer.kubeClient = kubeClient

	// ======== Create helm chart for created stacks and check helm release created ========
	stacks, err := s.db.ListPendingStacks()
	if err != nil {
		return err
	}
	for _, stack := range stacks {
		helmRelease, err := s.httpServer.findHelmRelease(ctx, stack.ToApi(), CurrentWorkspaceID)
		if err != nil {
			s.logger.Error(fmt.Sprintf("could not find helm release: %s", err.Error()))
			return fmt.Errorf("could not find helm release: %w", err)
		}
		s.logger.Info(fmt.Sprintf("Helm release creating for stack: %s", stack.StackID))
		if helmRelease == nil {
			if err := s.httpServer.createStackHelmRelease(ctx, CurrentWorkspaceID, stack.ToApi()); err != nil {
				s.logger.Error(fmt.Sprintf("failed to create helm release for stack: %s", stack.StackID), zap.Error(err))
				s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusFailed)
				s.db.UpdateStackFailureMessage(stack.StackID, fmt.Sprintf("failed to create helm release: %s", err.Error()))
			} else {
				s.logger.Error(fmt.Sprintf("helm release for stack %s not created", stack.StackID))
			}
		} else {
			if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
				s.logger.Info(fmt.Sprintf("Helm release created for stack: %s", stack.StackID))
				s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusCreated)
			} else if meta.IsStatusConditionFalse(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
				s.logger.Info(fmt.Sprintf("Helm release not ready for stack: %s", stack.StackID))
				if !helmRelease.Spec.Suspend {
					helmRelease.Spec.Suspend = true
					err = s.httpServer.kubeClient.Update(ctx, helmRelease)
					if err != nil {
						s.logger.Error(fmt.Sprintf("failed to suspend helmrelease for stack: %s", stack.StackID), zap.Error(err))
						s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusFailed)
						s.db.UpdateStackFailureMessage(stack.StackID, fmt.Sprintf("failed to suspend helmrelease: %s", err.Error()))
					}
				} else {
					helmRelease.Spec.Suspend = false
					err = s.httpServer.kubeClient.Update(ctx, helmRelease)
					if err != nil {
						s.logger.Error(fmt.Sprintf("failed to unsuspend helmrelease for stack: %s", stack.StackID), zap.Error(err))
						s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusFailed)
						s.db.UpdateStackFailureMessage(stack.StackID, fmt.Sprintf("failed to unsuspend helmrelease: %s", err.Error()))
					}
				}
			} else if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.StalledCondition) {
				s.logger.Info(fmt.Sprintf("Helm release stalled for stack: %s", stack.StackID))
				s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusStalled) // Temporary for debug
			}
		}
	}

	// ======== Run describer for created stacks ========
	stacks, err = s.db.ListCreatedStacks()
	for _, stack := range stacks {
		err = s.triggerStackDescriberJob(stack.ToApi())
		if err != nil {
			s.logger.Error(fmt.Sprintf("Failed to describe stack resources %s", stack.StackID), zap.Error(err))
			s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusFailed)
			s.db.UpdateStackFailureMessage(stack.StackID, fmt.Sprintf("Failed to describe stack resources with error: %s", err.Error()))
		} else {
			s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusDescribing)
		}
	}

	// ======== Check describer jobs and update stack status ========
	stacks, err = s.db.ListDescribingStacks()
	if err != nil {
		return err
	}
	for _, stack := range stacks {
		jobs, err := s.db.GetDescribeConnectionJobByConnectionID(stack.StackID)
		if err != nil {
			return err
		}
		if len(jobs) == 0 {
			continue
		} else {
			finished := true
			for _, job := range jobs {
				if job.Status == apiDescribe.DescribeResourceJobCreated ||
					job.Status == apiDescribe.DescribeResourceJobQueued ||
					job.Status == apiDescribe.DescribeResourceJobInProgress {
					finished = false
				}
			}

			if finished {
				s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusDescribed) // don't need to check sink. it waits one minutes
			}
		}
	}

	// ======== run evaluations on stacks ========
	stacks, err = s.db.ListDescribedStacks()
	if err != nil {
		return err
	}
	for _, stack := range stacks {
		s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusEvaluating)
		err = s.runStackBenchmarks(stack.ToApi())
		if err != nil {
			s.logger.Error(fmt.Sprintf("Failed to evaluate stack resources %s", stack.StackID), zap.Error(err))
			s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusFailed)
			s.db.UpdateStackFailureMessage(stack.StackID, fmt.Sprintf("Failed to run benchmarks on stack with error: %s", err.Error()))
		}
		err = s.runStackInsights(stack.ToApi())
		if err != nil {
			s.logger.Error(fmt.Sprintf("Failed to evaluate stack resources %s", stack.StackID), zap.Error(err))
			s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusFailed)
			s.db.UpdateStackFailureMessage(stack.StackID, fmt.Sprintf("Failed to run insights on stack with error: %s", err.Error()))
		}

	}

	// ======== Check evaluation jobs completed and remove helm release ========
	stacks, err = s.db.ListEvaluatingStacks()
	if err != nil {
		return err
	}
	for _, stack := range stacks {
		isComplete, err := s.updateStackJobs(stack.ToApi())
		if err != nil {
			return err
		}
		if isComplete {
			s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusCompleted)
			err = s.httpServer.deleteStackHelmRelease(stack.ToApi(), CurrentWorkspaceID)
			if err != nil {
				s.logger.Error(fmt.Sprintf("Failed to delete helmrelease for stack: %s", stack.StackID), zap.Error(err))
				s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusFailed)
				s.db.UpdateStackFailureMessage(stack.StackID, fmt.Sprintf("Failed to delete helmrelease: %s", err.Error()))
			}
		}
	}

	// ======== Delete failed helm releases ========
	stacks, err = s.db.ListFailedStacks()
	if err != nil {
		return err
	}
	for _, stack := range stacks {
		err = s.httpServer.deleteStackHelmRelease(stack.ToApi(), CurrentWorkspaceID)
		if err != nil {
			s.logger.Error(fmt.Sprintf("Failed to delete helmrelease for stack: %s", stack.StackID), zap.Error(err))
			s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusFailed)
			s.db.UpdateStackFailureMessage(stack.StackID, fmt.Sprintf("Failed to delete helmrelease: %s", err.Error()))
		} else {
			s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusCompletedWithFailure)
		}
	}
	return nil
}

func (s *Scheduler) triggerStackDescriberJob(stack apiDescribe.Stack) error {
	var provider source.Type
	for _, resource := range stack.Resources {
		if strings.Contains(resource, "aws") {
			provider = source.CloudAWS
		} else if strings.Contains(resource, "subscriptions") {
			provider = source.CloudAzure
		}
	}
	resourceTypes := stack.ResourceTypes
	rand.Shuffle(len(resourceTypes), func(i, j int) { resourceTypes[i], resourceTypes[j] = resourceTypes[j], resourceTypes[i] })
	for _, rType := range resourceTypes {
		describeResourceJob := DescribeConnectionJob{
			ConnectionID: stack.StackID,
			Connector:    source.Type(provider),
			AccountID:    stack.AccountIDs[0], // assume we have one account
			TriggerType:  enums.DescribeTriggerTypeStack,
			ResourceType: rType,
			Status:       apiDescribe.DescribeResourceJobCreated,
		}

		err := s.db.CreateDescribeConnectionJob(&describeResourceJob)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) storeStackCredentials(stack apiDescribe.Stack, configStr string) error {
	var provider source.Type
	for _, resource := range stack.Resources {
		if strings.Contains(resource, "aws") {
			provider = source.CloudAWS
		} else if strings.Contains(resource, "subscriptions") {
			provider = source.CloudAzure
		}
	}
	var secretBytes []byte
	kms, err := vault.NewKMSVaultSourceConfig(context.Background(), "", "", KeyRegion)
	if err != nil {
		return err
	}
	switch provider {
	case source.CloudAzure:
		config := apiOnboard.AzureCredentialConfig{}
		err := json.Unmarshal([]byte(configStr), &config)
		if err != nil {
			return fmt.Errorf("invalid config")
		}
		secretBytes, err = kms.Encrypt(config.AsMap(), KeyARN)
		if err != nil {
			return err
		}
	case source.CloudAWS:
		config := apiOnboard.AWSCredentialConfig{}
		err := json.Unmarshal([]byte(configStr), &config)
		if err != nil {
			return fmt.Errorf("invalid config")
		}
		secretBytes, err = kms.Encrypt(config.AsMap(), KeyARN)
		if err != nil {
			return err
		}
	}
	err = s.db.CreateStackCredential(&StackCredential{StackID: stack.StackID, Secret: string(secretBytes)})
	if err != nil {
		return err
	}
	return nil
}

func (s *Scheduler) runStackBenchmarks(stack apiDescribe.Stack) error {
	ctx := &httpclient.Context{
		UserRole: apiAuth.AdminRole,
	}
	benchmarks, err := s.complianceClient.ListBenchmarks(ctx)
	var provider source.Type
	for _, resource := range stack.Resources {
		if strings.Contains(resource, "aws") {
			provider = source.CloudAWS
		} else if strings.Contains(resource, "subscriptions") {
			provider = source.CloudAzure
		}
	}
	for _, benchmark := range benchmarks {
		connectorMatch := false
		for _, p := range benchmark.Connectors {
			if p == provider {
				connectorMatch = true
			}
		}
		if !connectorMatch { // pass if connector doesn't match
			continue
		}
		crj := newComplianceReportJob(stack.StackID, stack.SourceType, benchmark.ID)
		crj.IsStack = true

		err = s.db.CreateComplianceReportJob(&crj)
		if err != nil {
			return err
		}
		src := &apiOnboard.Connection{
			ConnectionID: stack.AccountIDs[0],
			Connector:    provider,
			Credential: apiOnboard.Credential{
				Config: "",
			},
		}
		enqueueComplianceReportJobs(s.logger, s.db, s.complianceReportJobQueue, *src, &crj)

		evaluation := StackEvaluation{
			EvaluatorID: benchmark.ID,
			Type:        api.EvaluationTypeBenchmark,
			StackID:     stack.StackID,
			JobID:       crj.ID,
			Status:      api.StackEvaluationStatusInProgress,
		}
		err = s.db.AddEvaluation(&evaluation)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) runStackInsights(stack apiDescribe.Stack) error {
	var provider source.Type
	for _, resource := range stack.Resources {
		if strings.Contains(resource, "aws") {
			provider = source.CloudAWS
		} else if strings.Contains(resource, "subscriptions") {
			provider = source.CloudAzure
		}
	}
	insights, err := s.complianceClient.ListInsightsMetadata(&httpclient.Context{UserRole: apiAuth.AdminRole}, []source.Type{provider})
	if err != nil {
		return err
	}
	for _, insight := range insights {
		job := newInsightJob(insight, stack.SourceType, stack.StackID, stack.AccountIDs[0], "")
		job.IsStack = true

		err = s.db.AddInsightJob(&job)
		if err != nil {
			return err
		}

		err = enqueueInsightJobs(s.insightJobQueue, job, insight)
		if err != nil {
			job.Status = apiInsight.InsightJobFailed
			job.FailureMessage = "Failed to enqueue InsightJob"
			s.db.UpdateInsightJobStatus(job)
		}
		evaluation := StackEvaluation{
			EvaluatorID: strconv.FormatUint(uint64(insight.ID), 10),
			Type:        api.EvaluationTypeInsight,
			StackID:     stack.StackID,
			JobID:       job.ID,
			Status:      api.StackEvaluationStatusInProgress,
		}
		err = s.db.AddEvaluation(&evaluation)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) updateStackJobs(stack apiDescribe.Stack) (bool, error) { // returns true if all jobs are completed
	isAllDone := true
	for _, evaluation := range stack.Evaluations {
		if evaluation.Status != apiDescribe.StackEvaluationStatusInProgress {
			continue
		}
		if evaluation.Type == api.EvaluationTypeBenchmark {
			job, err := s.db.GetComplianceReportJobByID(evaluation.JobID)
			if err != nil {
				return false, err
			}
			if job.Status == apiCompliance.ComplianceReportJobCompleted {
				err = s.db.UpdateEvaluationStatus(evaluation.JobID, apiDescribe.StackEvaluationStatusCompleted)
			} else if job.Status == apiCompliance.ComplianceReportJobCompletedWithFailure {
				err = s.db.UpdateEvaluationStatus(evaluation.JobID, apiDescribe.StackEvaluationStatusFailed)
			} else {
				isAllDone = false
			}
		} else if evaluation.Type == api.EvaluationTypeInsight {
			job, err := s.db.GetInsightJobById(evaluation.JobID)
			if err != nil {
				return false, err
			}
			if job.Status == apiInsight.InsightJobSucceeded {
				err = s.db.UpdateEvaluationStatus(evaluation.JobID, apiDescribe.StackEvaluationStatusCompleted)
			} else if job.Status == apiInsight.InsightJobFailed {
				err = s.db.UpdateEvaluationStatus(evaluation.JobID, apiDescribe.StackEvaluationStatusFailed)
			} else {
				isAllDone = false
			}
		}
	}
	return isAllDone, nil
}

func (s *Scheduler) getKafkaLag(topic string) (int, error) {
	err := s.kafkaConsumer.Subscribe(topic, nil)
	if err != nil {
		return 0, err
	}

	metadata, err := s.kafkaConsumer.GetMetadata(&topic, false, 5000)
	if err != nil {
		return 0, err
	}

	numPartitions := len(metadata.Topics[topic].Partitions)
	sum := 0
	for partition := 0; partition < numPartitions; partition++ {
		committed, err := s.kafkaConsumer.Committed([]kafka.TopicPartition{{Topic: &topic, Partition: int32(partition)}}, 5000)
		if err != nil {
			continue
		}

		_, high, err := s.kafkaConsumer.QueryWatermarkOffsets(topic, int32(partition), 5000)
		if err != nil {
			continue
		}

		offset := committed[0].Offset

		lag := high - int64(offset)
		sum = sum + int(lag)
	}
	return sum, nil
}
