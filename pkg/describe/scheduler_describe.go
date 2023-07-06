package describe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
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
	summarizerapi "github.com/kaytu-io/kaytu-engine/pkg/summarizer/api"
	"github.com/kaytu-io/kaytu-util/pkg/concurrency"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"go.uber.org/zap"
	"gorm.io/gorm"

	apimeta "github.com/fluxcd/pkg/apis/meta"
	"k8s.io/apimachinery/pkg/api/meta"
)

const (
	MaxQueued                       = 5000
	MaxAccountConcurrentQueued      = 10
	MaxResourceTypeConcurrentQueued = 50
)

type CloudNativeCall struct {
	dr  DescribeResourceJob
	ds  *DescribeSourceJob
	src *Source
}

func (s Scheduler) RunDescribeJobScheduler() {
	s.logger.Info("Scheduling describe jobs on a timer")

	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleDescribeJob()
		s.scheduleStackJobs()
	}
}

func (s Scheduler) RunDescribeResourceJobCycle() error {
	count, err := s.db.CountQueuedDescribeResourceJobs()
	if err != nil {
		s.logger.Error("failed to get queue length", zap.String("spot", "CountQueuedDescribeResourceJobs"), zap.Error(err))
		DescribeResourceJobsCount.WithLabelValues("failure").Inc()
		return err
	}

	if count > MaxQueued {
		s.logger.Error("queue is full", zap.String("spot", "count > MaxQueued"), zap.Error(err))
		return errors.New("queue is full")
	}

	drs, err := s.db.ListRandomCreatedDescribeResourceJobs(int(s.MaxConcurrentCall))
	if err != nil {
		s.logger.Error("failed to fetch describe resource jobs", zap.String("spot", "ListRandomCreatedDescribeResourceJobs"), zap.Error(err))
		DescribeResourceJobsCount.WithLabelValues("failure").Inc()
		return err
	}

	if len(drs) == 0 {
		if count == 0 {
			drs, err = s.db.GetFailedDescribeResourceJobs()
			if err != nil {
				s.logger.Error("failed to fetch failed describe resource jobs", zap.String("spot", "GetFailedDescribeResourceJobs"), zap.Error(err))
				DescribeResourceJobsCount.WithLabelValues("failure").Inc()
				return err
			}
			if len(drs) == 0 {
				return errors.New("no job to run")
			}
		} else {
			return errors.New("queue is not empty to look for retries")
		}
	}
	s.logger.Info("preparing resource jobs to run", zap.Int("length", len(drs)))

	parentMap := map[uint]*DescribeSourceJob{}
	srcMap := map[uint]*Source{}

	jobCount := 0

	wp := concurrency.NewWorkPool(len(drs))
	for _, dr := range drs {
		var ds *DescribeSourceJob
		var src *Source
		if v, ok := parentMap[dr.ParentJobID]; ok {
			ds = v
			src = srcMap[dr.ParentJobID]
		} else {
			ds, err = s.db.GetDescribeSourceJob(dr.ParentJobID)
			if err != nil {
				s.logger.Error("failed to get describe source job", zap.String("spot", "GetDescribeSourceJob"), zap.Error(err), zap.Uint("jobID", dr.ID))
				DescribeResourceJobsCount.WithLabelValues("failure").Inc()
				return err
			}
			if ds.TriggerType != enums.DescribeTriggerTypeStack {
				src, err = s.db.GetSourceByUUID(ds.SourceID)
				if err != nil {
					s.logger.Error("failed to get source", zap.String("spot", "GetSourceByUUID"), zap.Error(err), zap.Uint("jobID", dr.ID))
					DescribeResourceJobsCount.WithLabelValues("failure").Inc()
					return err
				}
				srcMap[dr.ParentJobID] = src
			}
			parentMap[dr.ParentJobID] = ds
		}

		if ds.TriggerType == enums.DescribeTriggerTypeStack {
			cred, err := s.db.GetStackCredential(ds.SourceID)
			if err != nil {
				return err
			}
			if cred.Secret == "" {
				return errors.New(fmt.Sprintf("No secret found for %s", ds.SourceID))
			}
			wp.AddJob(func() (interface{}, error) {
				err := s.enqueueCloudNativeDescribeJob(dr, ds, cred.Secret, s.WorkspaceName, ds.SourceID)
				if err != nil {
					s.logger.Error("Failed to enqueueCloudNativeDescribeConnectionJob", zap.Error(err), zap.Uint("jobID", dr.ID))
					DescribeResourceJobsCount.WithLabelValues("failure").Inc()
					return nil, err
				}
				DescribeResourceJobsCount.WithLabelValues("successful").Inc()
				return nil, nil
			})

		} else {
			c := CloudNativeCall{
				dr:  dr,
				ds:  ds,
				src: src,
			}
			wp.AddJob(func() (interface{}, error) {
				err := s.enqueueCloudNativeDescribeJob(c.dr, c.ds, c.src.ConfigRef, s.WorkspaceName, s.kafkaResourcesTopic)
				if err != nil {
					s.logger.Error("Failed to enqueueCloudNativeDescribeConnectionJob", zap.Error(err), zap.Uint("jobID", c.dr.ID))
					DescribeResourceJobsCount.WithLabelValues("failure").Inc()
					return nil, err
				}
				DescribeResourceJobsCount.WithLabelValues("successful").Inc()
				return nil, nil
			})
		}

		jobCount++
	}
	wp.Run()

	return nil
}

func (s Scheduler) RunDescribeResourceJobs() {
	for {
		if err := s.RunDescribeResourceJobCycle(); err != nil {
			time.Sleep(5 * time.Second)
		}
		time.Sleep(1 * time.Second)
	}
}

func (s Scheduler) scheduleDescribeJob() {
	err := s.CheckWorkspaceResourceLimit()
	if err != nil {
		s.logger.Error("failed to get limits", zap.String("spot", "CheckWorkspaceResourceLimit"), zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	connections, err := s.db.ListSources()
	if err != nil {
		s.logger.Error("failed to get list of sources", zap.String("spot", "ListSources"), zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}
	for _, connection := range connections {
		err = s.describeConnection(connection, true)
		if err != nil {
			s.logger.Error("failed to describe connection", zap.String("connection_id", connection.ID), zap.Error(err))
		}
	}

	DescribeJobsCount.WithLabelValues("successful").Inc()
}

func (s Scheduler) describeConnection(connection Source, scheduled bool) error {
	job, err := s.db.GetLastDescribeSourceJob(connection.ID)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		return err
	}

	if !scheduled || // manual
		job == nil || job.UpdatedAt.Before(time.Now().Add(time.Duration(-s.describeIntervalHours)*time.Hour)) {

		healthCheckedSrc, err := s.onboardClient.GetSourceHealthcheck(&httpclient.Context{
			UserRole: apiAuth.EditorRole,
		}, connection.ID)
		if err != nil {
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			return err
		}

		if scheduled && healthCheckedSrc.AssetDiscoveryMethod != source.AssetDiscoveryMethodTypeScheduled {
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			return errors.New("asset discovery is not scheduled")
		}

		if healthCheckedSrc.LifecycleState == apiOnboard.ConnectionLifecycleStateUnhealthy {
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			return errors.New("connection is not healthy")
		}

		describedAt := time.Now()
		triggerType := enums.DescribeTriggerTypeScheduled
		if healthCheckedSrc.LifecycleState == apiOnboard.ConnectionLifecycleStateInProgress {
			triggerType = enums.DescribeTriggerTypeInitialDiscovery
		}
		s.logger.Debug("Source is due for a describe. Creating a job now", zap.String("sourceId", connection.ID))

		fullDiscoveryJob, err := s.db.GetLastFullDiscoveryDescribeSourceJob(connection.ID)
		if err != nil {
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			return err
		}

		isFullDiscovery := false
		if job == nil ||
			fullDiscoveryJob == nil ||
			fullDiscoveryJob.UpdatedAt.Add(time.Duration(s.fullDiscoveryIntervalHours)*time.Hour).Before(time.Now()) {
			isFullDiscovery = true
		}
		daj := newDescribeSourceJob(connection, describedAt, triggerType, isFullDiscovery)
		err = s.db.CreateDescribeSourceJob(&daj)
		if err != nil {
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			return err
		}
		DescribeSourceJobsCount.WithLabelValues("successful").Inc()

		if healthCheckedSrc.LifecycleState == apiOnboard.ConnectionLifecycleStateInProgress {
			_, err = s.onboardClient.SetConnectionLifecycleState(&httpclient.Context{
				UserRole: apiAuth.EditorRole,
			}, connection.ID, apiOnboard.ConnectionLifecycleStateOnboard)
			if err != nil {
				s.logger.Warn("Failed to set connection lifecycle state", zap.String("connection_id", connection.ID), zap.Error(err))
			}
		}
	}
	return nil
}

func newDescribeSourceJob(a Source, describedAt time.Time, triggerType enums.DescribeTriggerType, isFullDiscovery bool) DescribeSourceJob {
	daj := DescribeSourceJob{
		DescribedAt:          describedAt,
		SourceID:             a.ID,
		SourceType:           a.Type,
		AccountID:            a.AccountID,
		DescribeResourceJobs: []DescribeResourceJob{},
		Status:               apiDescribe.DescribeSourceJobCreated,
		TriggerType:          triggerType,
		FullDiscovery:        isFullDiscovery,
	}
	var resourceTypes []string
	switch a.Type {
	case source.CloudAWS:
		if isFullDiscovery {
			resourceTypes = aws.ListResourceTypes()
		} else {
			resourceTypes = aws.ListFastDiscoveryResourceTypes()
		}
	case source.CloudAzure:
		if isFullDiscovery {
			resourceTypes = azure.ListResourceTypes()
		} else {
			resourceTypes = azure.ListFastDiscoveryResourceTypes()
		}
	default:
		panic(fmt.Errorf("unsupported source type: %s", a.Type))
	}

	rand.Shuffle(len(resourceTypes), func(i, j int) { resourceTypes[i], resourceTypes[j] = resourceTypes[j], resourceTypes[i] })
	for _, rType := range resourceTypes {
		daj.DescribeResourceJobs = append(daj.DescribeResourceJobs, DescribeResourceJob{
			ResourceType: rType,
			Status:       apiDescribe.DescribeResourceJobCreated,
		})
	}
	return daj
}

func (s Scheduler) enqueueCloudNativeDescribeJob(dr DescribeResourceJob, ds *DescribeSourceJob, cipherText string, workspaceName string, kafkaTopic string) error {
	s.logger.Debug("enqueueCloudNativeDescribeJob",
		zap.Uint("sourceJobID", ds.ID),
		zap.Uint("jobID", dr.ID),
		zap.String("connectionID", ds.SourceID),
		zap.String("resourceType", dr.ResourceType),
	)

	input := LambdaDescribeWorkerInput{
		WorkspaceId:      CurrentWorkspaceID,
		WorkspaceName:    workspaceName,
		DescribeEndpoint: s.describeEndpoint,
		KeyARN:           s.keyARN,
		KeyRegion:        s.keyRegion,
		KafkaTopic:       kafkaTopic,
		DescribeJob: DescribeJob{
			JobID:        dr.ID,
			ParentJobID:  ds.ID,
			ResourceType: dr.ResourceType,
			SourceID:     ds.SourceID,
			AccountID:    ds.AccountID,
			DescribedAt:  ds.DescribedAt.UnixMilli(),
			SourceType:   ds.SourceType,
			CipherText:   cipherText,
			TriggerType:  ds.TriggerType,
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
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/kaytu-%s-describer", LambdaFuncsBaseURL, strings.ToLower(ds.SourceType.String())), bytes.NewBuffer(lambdaRequest))
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
		zap.Uint("sourceJobID", ds.ID),
		zap.Uint("jobID", dr.ID),
		zap.String("connectionID", ds.SourceID),
		zap.String("resourceType", dr.ResourceType),
	)

	if err := s.db.QueueDescribeResourceJob(dr.ID); err != nil {
		s.logger.Error("failed to QueueDescribeResourceJob",
			zap.Uint("sourceJobID", ds.ID),
			zap.Uint("jobID", dr.ID),
			zap.String("connectionID", ds.SourceID),
			zap.String("resourceType", dr.ResourceType),
			zap.Error(err),
		)
	}
	return nil
}

// ================================================ STACKS ================================================

func (s Scheduler) scheduleStackJobs() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	kubeClient, err := s.httpServer.newKubeClient()
	if err != nil {
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
			return fmt.Errorf("could not find helm release: %w", err)
		}

		if helmRelease == nil {
			if err := s.httpServer.createStackHelmRelease(ctx, CurrentWorkspaceID, stack.ToApi()); err != nil {
				s.logger.Error(fmt.Sprintf("failed to create helm release for stack: %s", stack.StackID), zap.Error(err))
				s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusFailed)
				s.db.UpdateStackFailureMessage(stack.StackID, fmt.Sprintf("failed to create helm release: %s", err.Error()))
			}
		} else {
			if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
				s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusCreated)
			} else if meta.IsStatusConditionFalse(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
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
		jobs, err := s.db.QueryDescribeSourceJobs(stack.StackID)
		if err != nil {
			return err
		}
		if len(jobs) == 0 {
			continue
		} else {
			if jobs[0].Status == apiDescribe.DescribeSourceJobCompleted || jobs[0].Status == apiDescribe.DescribeSourceJobCompletedWithFailure {
				lag, err := s.getKafkaLag(stack.StackID)
				if err != nil {
					s.logger.Error(fmt.Sprintf("Failed to check kafka lag %s", stack.StackID), zap.Error(err))
					s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusFailed)
					s.db.UpdateStackFailureMessage(stack.StackID, fmt.Sprintf("Failed to check kafka lag with error: %s", err.Error()))
				}
				if lag == 0 {
					s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusDescribed)
				} else {
					s.db.UpdateStackFailureMessage(stack.StackID, fmt.Sprintf("Kafka lag is %d", lag))
				}
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
			// err = s.httpServer.deleteStackHelmRelease(stack.ToApi(), CurrentWorkspaceID)
			// if err != nil {
			// 	s.logger.Error(fmt.Sprintf("Failed to delete helmrelease for stack: %s", stack.StackID), zap.Error(err))
			// 	s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusFailed)
			// 	s.db.UpdateStackFailureMessage(stack.StackID, fmt.Sprintf("Failed to delete helmrelease: %s", err.Error()))
			// }
		}
	}

	// // ======== Delete failed helm releases ========
	// stacks, err = s.db.ListFailedStacks()
	// if err != nil {
	// 	return err
	// }
	// for _, stack := range stacks {
	// 	err = s.httpServer.deleteStackHelmRelease(stack.ToApi(), CurrentWorkspaceID)
	// 	if err != nil {
	// 		s.logger.Error(fmt.Sprintf("Failed to delete helmrelease for stack: %s", stack.StackID), zap.Error(err))
	// 		s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusFailed)
	// 		s.db.UpdateStackFailureMessage(stack.StackID, fmt.Sprintf("Failed to delete helmrelease: %s", err.Error()))
	// 	} else {
	// 		s.db.UpdateStackStatus(stack.StackID, apiDescribe.StackStatusCompletedWithFailure)
	// 	}
	// }
	return nil
}

func (s Scheduler) triggerStackDescriberJob(stack apiDescribe.Stack) error {
	var provider source.Type
	for _, resource := range stack.Resources {
		if strings.Contains(resource, "aws") {
			provider = source.CloudAWS
		} else if strings.Contains(resource, "subscriptions") {
			provider = source.CloudAzure
		}
	}
	describedAt := time.Now()
	resourceTypes := stack.ResourceTypes
	var describeResourceJobs []DescribeResourceJob
	rand.Shuffle(len(resourceTypes), func(i, j int) { resourceTypes[i], resourceTypes[j] = resourceTypes[j], resourceTypes[i] })
	for _, rType := range resourceTypes {
		describeResourceJobs = append(describeResourceJobs, DescribeResourceJob{
			ResourceType: rType,
			Status:       apiDescribe.DescribeResourceJobCreated,
		})
	}

	dsj := DescribeSourceJob{
		DescribedAt:          describedAt,
		SourceID:             stack.StackID,
		SourceType:           source.Type(provider),
		AccountID:            stack.AccountIDs[0], // assume we have one account
		DescribeResourceJobs: describeResourceJobs,
		Status:               apiDescribe.DescribeSourceJobCreated,
		TriggerType:          enums.DescribeTriggerTypeStack,
		FullDiscovery:        false,
	}

	err := s.db.CreateDescribeSourceJob(&dsj)
	if err != nil {
		return err
	}
	return nil
}

func (s Scheduler) storeStackCredentials(stack apiDescribe.Stack, configStr string) error {
	var provider source.Type
	for _, resource := range stack.Resources {
		if strings.Contains(resource, "aws") {
			provider = source.CloudAWS
		} else if strings.Contains(resource, "subscriptions") {
			provider = source.CloudAzure
		}
	}
	var secretBytes []byte
	kms, err := vault.NewKMSVaultSourceConfig(context.Background(), KMSAccessKey, KMSSecretKey, KeyRegion)
	if err != nil {
		return err
	}
	switch provider {
	case source.CloudAzure:
		config := apiOnboard.SourceConfigAzure{}
		err := json.Unmarshal([]byte(configStr), &config)
		if err != nil {
			return fmt.Errorf("invalid config")
		}
		secretBytes, err = kms.Encrypt(config.AsMap(), KeyARN)
		if err != nil {
			return err
		}
	case source.CloudAWS:
		config := apiOnboard.SourceConfigAWS{}
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

func (s Scheduler) runStackBenchmarks(stack apiDescribe.Stack) error {

	job := ScheduleJob{
		Model:          gorm.Model{},
		Status:         summarizerapi.SummarizerJobInProgress,
		FailureMessage: "",
	}
	err := s.db.AddScheduleJob(&job)
	if err != nil {
		return err
	}
	scheduleJob, err := s.db.FetchLastScheduleJob()
	if err != nil {
		return err
	}
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
		crj := newComplianceReportJob(stack.StackID, stack.SourceType, benchmark.ID, scheduleJob.ID)
		crj.IsStack = true

		err = s.db.CreateComplianceReportJob(&crj)
		if err != nil {
			return err
		}
		src := &Source{
			AccountID: stack.AccountIDs[0],
			Type:      provider,
			ConfigRef: "",
		}
		enqueueComplianceReportJobs(s.logger, s.db, s.complianceReportJobQueue, *src, &crj, scheduleJob)

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

func (s Scheduler) runStackInsights(stack apiDescribe.Stack) error {
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
		job := newInsightJob(insight, string(stack.SourceType), stack.StackID, stack.AccountIDs[0], "")
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

func (s Scheduler) updateStackJobs(stack apiDescribe.Stack) (bool, error) { // returns true if all jobs are completed
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

func (s Scheduler) getKafkaLag(topic string) (int, error) {
	//Get the assigned partitions.
	topicPartitions, err := s.kafkaConsumer.Assignment()
	if err != nil {
		return 0, err
	}

	// Get the offset for each partition assigned to this consumer instance
	topicPartitions, err = s.kafkaConsumer.Committed(topicPartitions, 5000)
	if err != nil {
		return 0, err
	}

	//Calculates the difference per partition of the consumer lag
	var l, highOffset int64
	for i := range topicPartitions {
		if *topicPartitions[i].Topic == topic {
			l, highOffset, err = s.kafkaConsumer.QueryWatermarkOffsets(*topicPartitions[i].Topic, topicPartitions[i].Partition, 5000)
			if err != nil {
				return 0, err
			}
			offset := int64(topicPartitions[i].Offset)
			if topicPartitions[i].Offset == kafka.OffsetInvalid {
				offset = l
			}
			//save information from the partition
			return int(highOffset - offset), nil
		}
	}
	return 0, fmt.Errorf("kafka topic %s not found", topic)
}
