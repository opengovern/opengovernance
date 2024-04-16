package describe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	apiDescribe "github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/config"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	apiOnboard "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-util/pkg/concurrency"
	"github.com/kaytu-io/kaytu-util/pkg/describe"
	"github.com/kaytu-io/kaytu-util/pkg/describe/enums"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/kaytu-io/kaytu-util/pkg/ticker"
	kaytuTrace "github.com/kaytu-io/kaytu-util/pkg/trace"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

const (
	MaxQueued      = 5000
	MaxIn10Minutes = 5000
)

var ErrJobInProgress = errors.New("job already in progress")

type CloudNativeCall struct {
	dc  model.DescribeConnectionJob
	src *apiOnboard.Connection
}

func (s *Scheduler) RunDescribeJobScheduler(ctx context.Context) {
	s.logger.Info("Scheduling describe jobs on a timer")

	t := ticker.NewTicker(60*time.Second, time.Second*10)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleDescribeJob(ctx)
	}
}

func (s *Scheduler) RunDescribeResourceJobCycle(ctx context.Context) error {
	ctx, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx, kaytuTrace.GetCurrentFuncName())
	defer span.End()

	if s.WorkspaceName == "" {
		return errors.New("workspace name is empty")
	}

	count, err := s.db.CountQueuedDescribeConnectionJobs()
	if err != nil {
		s.logger.Error("failed to get queue length", zap.String("spot", "CountQueuedDescribeConnectionJobs"), zap.Error(err))
		DescribeResourceJobsCount.WithLabelValues("failure", "queue_length").Inc()
		return err
	}

	if count > MaxQueued {
		DescribePublishingBlocked.WithLabelValues("cloud queued").Set(1)
		s.logger.Error("queue is full", zap.String("spot", "count > MaxQueued"), zap.Error(err))
		return errors.New("queue is full")
	} else {
		DescribePublishingBlocked.WithLabelValues("cloud queued").Set(0)
	}

	count, err = s.db.CountDescribeConnectionJobsRunOverLast10Minutes()
	if err != nil {
		s.logger.Error("failed to get last hour length", zap.String("spot", "CountDescribeConnectionJobsRunOverLastHour"), zap.Error(err))
		DescribeResourceJobsCount.WithLabelValues("failure", "last_hour_length").Inc()
		return err
	}

	if count > MaxIn10Minutes {
		DescribePublishingBlocked.WithLabelValues("hour queued").Set(1)
		s.logger.Error("too many jobs at last hour", zap.String("spot", "count > MaxQueued"), zap.Error(err))
		return errors.New("too many jobs at last hour")
	} else {
		DescribePublishingBlocked.WithLabelValues("hour queued").Set(0)
	}

	dcs, err := s.db.ListRandomCreatedDescribeConnectionJobs(ctx, int(s.MaxConcurrentCall))
	if err != nil {
		s.logger.Error("failed to fetch describe resource jobs", zap.String("spot", "ListRandomCreatedDescribeResourceJobs"), zap.Error(err))
		DescribeResourceJobsCount.WithLabelValues("failure", "fetch_error").Inc()
		return err
	}
	s.logger.Info("got the jobs", zap.Int("length", len(dcs)), zap.Int("limit", int(s.MaxConcurrentCall)))

	counts, err := s.db.CountRunningDescribeJobsPerResourceType()
	if err != nil {
		s.logger.Error("failed to resource type count", zap.String("spot", "CountRunningDescribeJobsPerResourceType"), zap.Error(err))
		DescribeResourceJobsCount.WithLabelValues("failure", "resource_type_count").Inc()
		return err
	}

	rand.Shuffle(len(dcs), func(i, j int) {
		dcs[i], dcs[j] = dcs[j], dcs[i]
	})

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
			src, err = s.onboardClient.GetSource(&httpclient.Context{UserRole: apiAuth.InternalRole}, dc.ConnectionID)
			if err != nil {
				s.logger.Error("failed to get source", zap.String("spot", "GetSourceByUUID"), zap.Error(err), zap.Uint("jobID", dc.ID))
				DescribeResourceJobsCount.WithLabelValues("failure", "get_source").Inc()
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
		c := CloudNativeCall{
			dc:  dc,
			src: src,
		}
		wp.AddJob(func() (interface{}, error) {
			err := s.enqueueCloudNativeDescribeJob(ctx, c.dc, c.src.Credential.Config.(string), s.WorkspaceName)
			if err != nil {
				s.logger.Error("Failed to enqueueCloudNativeDescribeConnectionJob", zap.Error(err), zap.Uint("jobID", dc.ID))
				DescribeResourceJobsCount.WithLabelValues("failure", "enqueue").Inc()
				return nil, err
			}
			DescribeResourceJobsCount.WithLabelValues("successful", "").Inc()
			return nil, nil
		})
	}

	res := wp.Run()
	for _, r := range res {
		if r.Error != nil {
			s.logger.Error("failure on calling cloudNative describer", zap.Error(r.Error))
		}
	}

	return nil
}

func (s *Scheduler) RunDescribeResourceJobs(ctx context.Context) {
	t := ticker.NewTicker(time.Second*30, time.Second*10)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if err := s.RunDescribeResourceJobCycle(ctx); err != nil {
				s.logger.Error("failure while RunDescribeResourceJobCycle", zap.Error(err))
			}
			t.Reset(time.Second*30, time.Second*10)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) scheduleDescribeJob(ctx context.Context) {
	//err := s.CheckWorkspaceResourceLimit()
	//if err != nil {
	//	s.logger.Error("failed to get limits", zap.String("spot", "CheckWorkspaceResourceLimit"), zap.Error(err))
	//	DescribeJobsCount.WithLabelValues("failure").Inc()
	//	return
	//}
	//
	s.logger.Info("running describe job scheduler")
	connections, err := s.onboardClient.ListSources(&httpclient.Context{UserRole: apiAuth.InternalRole}, nil)
	if err != nil {
		s.logger.Error("failed to get list of sources", zap.String("spot", "ListSources"), zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	rts, err := s.ListDiscoveryResourceTypes()
	if err != nil {
		s.logger.Error("failed to get list of resource types", zap.String("spot", "ListDiscoveryResourceTypes"), zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	for _, connection := range connections {
		s.logger.Info("running describe job scheduler for connection", zap.String("connection_id", connection.ID.String()))
		var resourceTypes []string
		switch connection.Connector {
		case source.CloudAWS:
			awsRts := aws.GetResourceTypesMap()
			for _, rt := range rts.AWSResourceTypes {
				if _, ok := awsRts[rt]; ok {
					resourceTypes = append(resourceTypes, rt)
				}
			}
		case source.CloudAzure:
			azureRts := azure.GetResourceTypesMap()
			for _, rt := range rts.AzureResourceTypes {
				if _, ok := azureRts[rt]; ok {
					resourceTypes = append(resourceTypes, rt)
				}
			}
		}

		for _, resourceType := range resourceTypes {
			if !connection.GetSupportedResourceTypeMap()[strings.ToLower(resourceType)] {
				s.logger.Warn("resource type is not supported on this connection, skipping describe", zap.String("connection_id", connection.ID.String()), zap.String("resource_type", resourceType))
				continue
			}

			removeResourcesAzure := azureAdOnlyOnOneConnection(connections, connection, resourceType)
			removeResourcesAWS := awsOnlyOnOneConnection(connections, connection, resourceType)
			_, err = s.describe(connection, resourceType, true, false, removeResourcesAzure || removeResourcesAWS)
			if err != nil {
				s.logger.Error("failed to describe connection", zap.String("connection_id", connection.ID.String()), zap.String("resource_type", resourceType), zap.Error(err))
			}
		}

		if connection.LifecycleState == apiOnboard.ConnectionLifecycleStateInProgress {
			_, err = s.onboardClient.SetConnectionLifecycleState(&httpclient.Context{
				UserRole: apiAuth.EditorRole,
			}, connection.ID.String(), apiOnboard.ConnectionLifecycleStateOnboard)
			if err != nil {
				s.logger.Warn("Failed to set connection lifecycle state", zap.String("connection_id", connection.ID.String()), zap.Error(err))
			}
		}
	}

	if err := s.retryFailedJobs(ctx); err != nil {
		s.logger.Error("failed to retry failed jobs", zap.String("spot", "retryFailedJobs"), zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	DescribeJobsCount.WithLabelValues("successful").Inc()
}

func azureAdOnlyOnOneConnection(connections []apiOnboard.Connection, connection apiOnboard.Connection, resourceType string) bool {
	if connection.Connector != source.CloudAzure {
		return false
	}

	if steampipe.ExtractPlugin(resourceType) != steampipe.SteampipePluginAzureAD {
		return false
	}

	connectionTenantID := connection.TenantID()
	if connectionTenantID != "" {
		var connectionIDs []string
		for _, c := range connections {
			if c.HealthState == source.HealthStatusUnhealthy {
				continue
			}

			if c.TenantID() == connectionTenantID {
				connectionIDs = append(connectionIDs, c.ID.String())
			}
		}

		sort.Strings(connectionIDs)

		if connection.ID.String() != connectionIDs[0] {
			return true
		}
	}
	return false
}

func awsOnlyOnOneConnection(connections []apiOnboard.Connection, connection apiOnboard.Connection, resourceType string) bool {
	if connection.Connector != source.CloudAWS {
		return false
	}

	if connection.CredentialType != apiOnboard.CredentialTypeManualAwsOrganization {
		return false
	}

	if !strings.HasPrefix(resourceType, "AWS::IdentityStore::") {
		return false
	}

	var AccountType string
	if accountType, ok := connection.Metadata["account_type"]; ok {
		if accountTypeStr, ok := accountType.(string); ok {
			AccountType = accountTypeStr
		}
	}

	if AccountType == "" {
		return false
	}

	return AccountType != "organization_manager"
}

func (s *Scheduler) retryFailedJobs(ctx context.Context) error {

	ctx, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx, "GetFailedJobs")
	defer span.End()

	fdcs, err := s.db.GetFailedDescribeConnectionJobs(ctx)
	if err != nil {
		s.logger.Error("failed to fetch failed describe resource jobs", zap.String("spot", "GetFailedDescribeResourceJobs"), zap.Error(err))
		return err
	}
	s.logger.Info(fmt.Sprintf("found %v failed jobs before filtering", len(fdcs)))
	retryCount := 0

	for _, failedJob := range fdcs {
		var isFastDiscovery, isCostDiscovery bool

		switch failedJob.Connector {
		case source.CloudAWS:
			resourceType, err := aws.GetResourceType(failedJob.ResourceType)
			if err != nil {
				return fmt.Errorf("failed to get aws resource type due to: %v", err)
			}
			isFastDiscovery, isCostDiscovery = resourceType.FastDiscovery, resourceType.CostDiscovery
		case source.CloudAzure:
			resourceType, err := azure.GetResourceType(failedJob.ResourceType)
			if err != nil {
				return fmt.Errorf("failed to get aws resource type due to: %v", err)
			}
			isFastDiscovery, isCostDiscovery = resourceType.FastDiscovery, resourceType.CostDiscovery
		}

		describeCycle := s.fullDiscoveryIntervalHours
		if isFastDiscovery {
			describeCycle = s.describeIntervalHours
		} else if isCostDiscovery {
			describeCycle = s.costDiscoveryIntervalHours
		}

		if failedJob.CreatedAt.Before(time.Now().Add(-1 * describeCycle)) {
			continue
		}

		err = s.db.RetryDescribeConnectionJob(failedJob.ID)
		if err != nil {
			return err
		}

		retryCount++
	}

	s.logger.Info(fmt.Sprintf("retrying %v failed jobs", retryCount))
	span.End()
	return nil
}

func (s *Scheduler) describe(connection apiOnboard.Connection, resourceType string, scheduled bool, costFullDiscovery bool, removeResources bool) (*model.DescribeConnectionJob, error) {
	if connection.CredentialType == apiOnboard.CredentialTypeManualAwsOrganization &&
		strings.HasPrefix(strings.ToLower(resourceType), "aws::costexplorer") {
		// cost on org
	} else {
		if !connection.IsEnabled() {
			return nil, nil
		}
	}

	job, err := s.db.GetLastDescribeConnectionJob(connection.ID.String(), resourceType)
	if err != nil {
		s.logger.Error("failed to get last describe job", zap.String("resource_type", resourceType), zap.String("connection_id", connection.ID.String()), zap.Error(err))
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		return nil, err
	}

	discoveryType := model.DiscoveryType_Full
	if connection.Connector == source.CloudAWS {
		rt, _ := aws.GetResourceType(resourceType)
		if rt != nil {
			if rt.FastDiscovery {
				discoveryType = model.DiscoveryType_Fast
			} else if rt.CostDiscovery {
				discoveryType = model.DiscoveryType_Cost
			}
		}
	} else if connection.Connector == source.CloudAzure {
		rt, _ := azure.GetResourceType(resourceType)
		if rt != nil {
			if rt.FastDiscovery {
				discoveryType = model.DiscoveryType_Fast
			} else if rt.CostDiscovery {
				discoveryType = model.DiscoveryType_Cost
			}
		}
	}

	if job != nil {
		if scheduled {
			interval := s.fullDiscoveryIntervalHours
			if connection.Connector == source.CloudAWS {
				rt, _ := aws.GetResourceType(resourceType)
				if rt != nil {
					if rt.FastDiscovery {
						discoveryType = model.DiscoveryType_Fast
						interval = s.describeIntervalHours
					} else if rt.CostDiscovery {
						discoveryType = model.DiscoveryType_Cost
						interval = s.costDiscoveryIntervalHours
					}
				}
			} else if connection.Connector == source.CloudAzure {
				rt, _ := azure.GetResourceType(resourceType)
				if rt != nil {
					if rt.FastDiscovery {
						discoveryType = model.DiscoveryType_Fast
						interval = s.describeIntervalHours
					} else if rt.CostDiscovery {
						discoveryType = model.DiscoveryType_Cost
						interval = s.costDiscoveryIntervalHours
					}
				}
			}

			if job.UpdatedAt.After(time.Now().Add(-interval)) {
				return nil, nil
			}
		}

		if job.Status == api.DescribeResourceJobCreated ||
			job.Status == api.DescribeResourceJobQueued ||
			job.Status == api.DescribeResourceJobInProgress ||
			job.Status == api.DescribeResourceJobOldResourceDeletion {
			return nil, ErrJobInProgress
		}
	}

	if connection.LastHealthCheckTime.Before(time.Now().Add(-1 * 24 * time.Hour)) {
		healthCheckedSrc, err := s.onboardClient.GetSourceHealthcheck(&httpclient.Context{
			UserRole: apiAuth.EditorRole,
		}, connection.ID.String(), false)
		if err != nil {
			s.logger.Error("failed to get source healthcheck", zap.String("resource_type", resourceType), zap.String("connection_id", connection.ID.String()), zap.Error(err))
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			return nil, err
		}
		connection = *healthCheckedSrc
	}

	if scheduled && connection.AssetDiscoveryMethod != source.AssetDiscoveryMethodTypeScheduled {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		return nil, errors.New("asset discovery is not scheduled")
	}

	if connection.CredentialType == apiOnboard.CredentialTypeManualAwsOrganization &&
		strings.HasPrefix(strings.ToLower(resourceType), "aws::costexplorer") {
		// cost on org
	} else {
		if (connection.LifecycleState != apiOnboard.ConnectionLifecycleStateOnboard &&
			connection.LifecycleState != apiOnboard.ConnectionLifecycleStateInProgress) ||
			connection.HealthState != source.HealthStatusHealthy {
			// DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			// return errors.New("connection is not healthy or disabled")
			return nil, nil
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
	daj := newDescribeConnectionJob(connection, resourceType, triggerType, discoveryType)
	if removeResources {
		daj.Status = apiDescribe.DescribeResourceJobRemovingResources
	}
	err = s.db.CreateDescribeConnectionJob(&daj)
	if err != nil {
		s.logger.Error("failed to create describe resource job", zap.String("resource_type", resourceType), zap.String("connection_id", connection.ID.String()), zap.Error(err))
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		return nil, err
	}
	DescribeSourceJobsCount.WithLabelValues("successful").Inc()

	return &daj, nil
}

func newDescribeConnectionJob(a apiOnboard.Connection, resourceType string, triggerType enums.DescribeTriggerType, discoveryType model.DiscoveryType) model.DescribeConnectionJob {
	return model.DescribeConnectionJob{
		ConnectionID:  a.ID.String(),
		Connector:     a.Connector,
		AccountID:     a.ConnectionID,
		TriggerType:   triggerType,
		ResourceType:  resourceType,
		Status:        apiDescribe.DescribeResourceJobCreated,
		DiscoveryType: discoveryType,
	}
}

func (s *Scheduler) enqueueCloudNativeDescribeJob(ctx context.Context, dc model.DescribeConnectionJob, cipherText string, workspaceName string) error {
	ctx, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx, kaytuTrace.GetCurrentFuncName())
	defer span.End()

	s.logger.Debug("enqueueCloudNativeDescribeJob",
		zap.Uint("jobID", dc.ID),
		zap.String("connectionID", dc.ConnectionID),
		zap.String("resourceType", dc.ResourceType),
	)

	input := describe.DescribeWorkerInput{
		WorkspaceId:               CurrentWorkspaceID,
		WorkspaceName:             workspaceName,
		DescribeEndpoint:          s.describeEndpoint,
		IngestionPipelineEndpoint: s.conf.ElasticSearch.IngestionEndpoint,
		UseOpenSearch:             s.conf.ElasticSearch.IsOpenSearch,

		VaultConfig: s.conf.Vault,

		DescribeJob: describe.DescribeJob{
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

	if err := s.db.QueueDescribeConnectionJob(dc.ID); err != nil {
		s.logger.Error("failed to QueueDescribeResourceJob",
			zap.Uint("jobID", dc.ID),
			zap.String("connectionID", dc.ConnectionID),
			zap.String("resourceType", dc.ResourceType),
			zap.Error(err),
		)
	}
	isFailed := false
	defer func() {
		if isFailed {
			err := s.db.UpdateDescribeConnectionJobStatus(dc.ID, apiDescribe.DescribeResourceJobFailed, "Failed to invoke lambda", "Failed to invoke lambda", 0, 0)
			if err != nil {
				s.logger.Error("failed to update describe resource job status",
					zap.Uint("jobID", dc.ID),
					zap.String("connectionID", dc.ConnectionID),
					zap.String("resourceType", dc.ResourceType),
					zap.Error(err),
				)
			}
		}
	}()

	switch s.conf.ServerlessProvider {
	case config.ServerlessProviderTypeAWSLambda.String():
		lambdaPayload, err := json.Marshal(input)
		if err != nil {
			s.logger.Error("failed to marshal cloud native req", zap.Uint("jobID", dc.ID), zap.String("connectionID", dc.ConnectionID), zap.String("resourceType", dc.ResourceType), zap.Error(err))
			return fmt.Errorf("failed to marshal cloud native req due to %w", err)
		}
		invokeOutput, err := s.lambdaClient.Invoke(ctx, &lambda.InvokeInput{
			FunctionName:   awsSdk.String(fmt.Sprintf("kaytu-%s-describer", strings.ToLower(dc.Connector.String()))),
			LogType:        types.LogTypeTail,
			Payload:        lambdaPayload,
			InvocationType: types.InvocationTypeEvent,
		})
		if err != nil {
			s.logger.Error("failed to invoke lambda function",
				zap.Uint("jobID", dc.ID),
				zap.String("connectionID", dc.ConnectionID),
				zap.String("resourceType", dc.ResourceType),
				zap.Error(err),
			)
			isFailed = true
			return fmt.Errorf("failed to invoke lambda function due to %v", err)
		}

		if invokeOutput.FunctionError != nil {
			s.logger.Info("lambda function function error",
				zap.String("resourceType", dc.ResourceType), zap.String("error", *invokeOutput.FunctionError))
		}
		if invokeOutput.LogResult != nil {
			s.logger.Info("lambda function log result",
				zap.String("resourceType", dc.ResourceType), zap.String("log result", *invokeOutput.LogResult))
		}

		s.logger.Info("lambda function payload",
			zap.String("resourceType", dc.ResourceType), zap.String("payload", fmt.Sprintf("%v", invokeOutput.Payload)))
		resBody := invokeOutput.Payload

		if invokeOutput.StatusCode == http.StatusTooManyRequests {
			s.logger.Error("failed to trigger cloud native worker due to too many requests", zap.Uint("jobID", dc.ID), zap.String("connectionID", dc.ConnectionID), zap.String("resourceType", dc.ResourceType))
			isFailed = true
			return fmt.Errorf("failed to trigger cloud native worker due to %d: %s", invokeOutput.StatusCode, string(resBody))
		}

		if invokeOutput.StatusCode != http.StatusAccepted {
			s.logger.Error("failed to trigger cloud native worker", zap.Uint("jobID", dc.ID), zap.String("connectionID", dc.ConnectionID), zap.String("resourceType", dc.ResourceType))
			isFailed = true
			return fmt.Errorf("failed to trigger cloud native worker due to %d: %s", invokeOutput.StatusCode, string(resBody))
		}
	case config.ServerlessProviderTypeAzureFunctions.String():
		input.DescribeEndpoint = s.describeExternalEndpoint
		eventHubPayload, err := json.Marshal(input)
		if err != nil {
			s.logger.Error("failed to marshal cloud native req", zap.Uint("jobID", dc.ID), zap.String("connectionID", dc.ConnectionID), zap.String("resourceType", dc.ResourceType), zap.Error(err))
			isFailed = true
			return fmt.Errorf("failed to marshal cloud native req due to %w", err)
		}
		sender, err := s.serviceBusClient.NewSender(fmt.Sprintf("kaytu-%s-describer", strings.ToLower(dc.Connector.String())), nil)
		if err != nil {
			s.logger.Error("failed to create service bus sender",
				zap.Uint("jobID", dc.ID),
				zap.String("connectionID", dc.ConnectionID),
				zap.String("resourceType", dc.ResourceType),
				zap.Error(err),
			)
			isFailed = true
			return fmt.Errorf("failed to create service bus sender due to %v", err)
		}
		defer sender.Close(ctx)
		err = sender.SendMessage(ctx, &azservicebus.Message{
			Body:        eventHubPayload,
			ContentType: utils.GetPointer("application/json"),
		}, nil)
		if err != nil {
			s.logger.Error("failed to send message to service bus",
				zap.Uint("jobID", dc.ID),
				zap.String("connectionID", dc.ConnectionID),
				zap.String("resourceType", dc.ResourceType),
				zap.Error(err),
			)
			isFailed = true
			return fmt.Errorf("failed to send message to service bus due to %v", err)
		}
		err = sender.Close(ctx)
		if err != nil {
			s.logger.Error("failed to close service bus sender",
				zap.Uint("jobID", dc.ID),
				zap.String("connectionID", dc.ConnectionID),
				zap.String("resourceType", dc.ResourceType),
				zap.Error(err),
			)
			isFailed = true
			return fmt.Errorf("failed to close service bus sender due to %v", err)
		}
	default:
		s.logger.Error("unknown serverless provider", zap.String("provider", s.conf.ServerlessProvider))
		isFailed = true
		return fmt.Errorf("unknown serverless provider: %s", s.conf.ServerlessProvider)
	}

	s.logger.Info("successful job trigger",
		zap.Uint("jobID", dc.ID),
		zap.String("connectionID", dc.ConnectionID),
		zap.String("resourceType", dc.ResourceType),
	)

	return nil
}
