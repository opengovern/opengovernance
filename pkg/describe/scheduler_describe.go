package describe

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	api2 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"go.uber.org/zap"
)

const (
	MaxTriggerPerMinute           = 5000
	MaxTriggerPerAccountPerMinute = 60
)

func (s Scheduler) RunDescribeJobScheduler() {
	s.logger.Info("Scheduling describe jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleDescribeJob()
	}
}

func (s Scheduler) scheduleDescribeJob() {
	s.logger.Info("scheduleDescribeJob")
	err := s.CheckWorkspaceResourceLimit()
	if err != nil {
		s.logger.Error("failure on CheckWorkspaceResourceLimit", zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	connections, err := s.db.ListSources()
	if err != nil {
		s.logger.Error("Failed to fetch all connections", zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}
	for _, connection := range connections {
		err = s.describeConnection(connection, true)
		if err != nil {
			s.logger.Error("Failed to describe connection", zap.String("connection_id", connection.ID.String()), zap.Error(err))
		}
	}

	err = s.db.RetryRateLimitedJobs()
	if err != nil {
		s.logger.Error("Failed to RetryRateLimitedJobs", zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	accountTriggerCount := map[string]int{}
	var parentIdExceptionList []uint
	for i := 0; i < MaxTriggerPerMinute; i++ {
		drs, err := s.db.FetchRandomCreatedDescribeResourceJobs(parentIdExceptionList)
		if err != nil {
			s.logger.Error("Failed to fetch all describe source jobs", zap.Error(err))
			DescribeJobsCount.WithLabelValues("failure").Inc()
			return
		}

		if drs == nil {
			break
		}

		ds, err := s.db.GetDescribeSourceJob(drs.ParentJobID)
		if err != nil {
			s.logger.Error("Failed to GetDescribeSourceJob in scheduler", zap.Error(err), zap.Uint("jobID", drs.ID))
			DescribeJobsCount.WithLabelValues("failure").Inc()
			DescribeResourceJobsCount.WithLabelValues("failure").Inc()
			return
		}

		if accountTriggerCount[ds.SourceID.String()] > MaxTriggerPerAccountPerMinute {
			parentIdExceptionList = append(parentIdExceptionList, drs.ParentJobID)
			continue
		}

		err = s.enqueueCloudNativeDescribeJob(*drs, ds)
		if err != nil {
			s.logger.Error("Failed to enqueueCloudNativeDescribeConnectionJob", zap.Error(err), zap.Uint("jobID", drs.ID))
			DescribeJobsCount.WithLabelValues("failure").Inc()
			DescribeResourceJobsCount.WithLabelValues("failure").Inc()
			return
		}
		accountTriggerCount[ds.SourceID.String()]++
		DescribeResourceJobsCount.WithLabelValues("successful").Inc()
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
			UserRole: api2.EditorRole,
		}, connection.ID.String())
		if err != nil {
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			return err
		}

		if scheduled && healthCheckedSrc.AssetDiscoveryMethod != source.AssetDiscoveryMethodTypeScheduled {
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			return errors.New("asset discovery is not scheduled")
		}

		if healthCheckedSrc.HealthState == source.HealthStatusUnhealthy {
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			return errors.New("connection is not healthy")
		}

		describedAt := time.Now()
		triggerType := enums.DescribeTriggerTypeScheduled
		if job == nil {
			triggerType = enums.DescribeTriggerTypeInitialDiscovery
		}

		s.logger.Info("Source is due for a describe. Creating a job now", zap.String("sourceId", connection.ID.String()))
		daj := newDescribeSourceJob(connection, describedAt, triggerType)
		err = s.db.CreateDescribeSourceJob(&daj)
		if err != nil {
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			return err
		}
		DescribeSourceJobsCount.WithLabelValues("successful").Inc()
	}
	return nil
}

func newDescribeSourceJob(a Source, describedAt time.Time, triggerType enums.DescribeTriggerType) DescribeSourceJob {
	daj := DescribeSourceJob{
		DescribedAt:          describedAt,
		SourceID:             a.ID,
		SourceType:           a.Type,
		AccountID:            a.AccountID,
		DescribeResourceJobs: []DescribeResourceJob{},
		Status:               api.DescribeSourceJobCreated,
		TriggerType:          triggerType,
	}
	var resourceTypes []string
	switch a.Type {
	case source.CloudAWS:
		resourceTypes = aws.ListResourceTypes()
	case source.CloudAzure:
		resourceTypes = azure.ListResourceTypes()
	default:
		panic(fmt.Errorf("unsupported source type: %s", a.Type))
	}

	rand.Shuffle(len(resourceTypes), func(i, j int) { resourceTypes[i], resourceTypes[j] = resourceTypes[j], resourceTypes[i] })
	for _, rType := range resourceTypes {
		daj.DescribeResourceJobs = append(daj.DescribeResourceJobs, DescribeResourceJob{
			ResourceType: rType,
			Status:       api.DescribeResourceJobCreated,
		})
	}
	return daj
}

func (s Scheduler) enqueueCloudNativeDescribeJob(dr DescribeResourceJob, ds *DescribeSourceJob) error {
	s.logger.Info("enqueueCloudNativeDescribeJob",
		zap.Uint("sourceJobID", ds.ID),
		zap.Uint("jobID", dr.ID),
		zap.String("connectionID", ds.SourceID.String()),
		zap.String("resourceType", dr.ResourceType),
	)

	if ds.Status == api.DescribeSourceJobCreated {
		err := s.db.UpdateDescribeSourceJob(ds.ID, api.DescribeSourceJobInProgress)
		if err != nil {
			return err
		}
	}

	src, err := s.db.GetSourceByUUID(ds.SourceID)
	if err != nil {
		return err
	}

	workspace, err := s.workspaceClient.GetByID(&httpclient.Context{
		UserRole: api2.EditorRole,
	}, CurrentWorkspaceID)
	if err != nil {
		return err
	}

	input := LambdaDescribeWorkerInput{
		WorkspaceId:      CurrentWorkspaceID,
		WorkspaceName:    workspace.Name,
		DescribeEndpoint: s.describeEndpoint,
		KeyARN:           s.keyARN,
		KeyRegion:        s.keyRegion,
		DescribeJob: DescribeJob{
			JobID:        dr.ID,
			ParentJobID:  ds.ID,
			ResourceType: dr.ResourceType,
			SourceID:     ds.SourceID.String(),
			AccountID:    ds.AccountID,
			DescribedAt:  ds.DescribedAt.UnixMilli(),
			SourceType:   ds.SourceType,
			CipherText:   src.ConfigRef,
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
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s", LambdaFuncsBaseURL, strings.ToLower(ds.SourceType.String())), bytes.NewBuffer(lambdaRequest))
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

	s.logger.Info("Successful job trigger",
		zap.Uint("sourceJobID", ds.ID),
		zap.Uint("jobID", dr.ID),
		zap.String("connectionID", ds.SourceID.String()),
		zap.String("resourceType", dr.ResourceType),
	)

	if err := s.db.UpdateDescribeResourceJobStatus(dr.ID, api.DescribeResourceJobQueued, "", 0); err != nil {
		s.logger.Error("Failed to update DescribeResourceJob",
			zap.Uint("sourceJobID", ds.ID),
			zap.Uint("jobID", dr.ID),
			zap.String("connectionID", ds.SourceID.String()),
			zap.String("resourceType", dr.ResourceType),
			zap.Error(err),
		)
	}
	return nil
}
