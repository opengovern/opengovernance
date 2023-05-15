package describe

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
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
	MaxTriggerPerMinute           = 10000
	MaxTriggerPerAccountPerMinute = 60
	MaxQueued                     = 10000
	MaxConcurrentCall             = 100
)

type CloudNativeCall struct {
	dr DescribeResourceJob
	ds *DescribeSourceJob
}

func (s Scheduler) RunDescribeJobScheduler() {
	s.logger.Info("Scheduling describe jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	s.cloudNativeCallChannel = make(chan CloudNativeCall, MaxConcurrentCall*2)
	for i := 0; i < MaxConcurrentCall; i++ {
		go s.cloudNativeCaller()
	}

	for ; ; <-t.C {
		s.scheduleDescribeJob()
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
			s.logger.Error("failed to describe connection", zap.String("connection_id", connection.ID.String()), zap.Error(err))
		}
	}

	err = s.db.RetryRateLimitedJobs()
	if err != nil {
		s.logger.Error("failed to update database", zap.String("spot", "RetryRateLimitedJobs"), zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	accountTriggerCount := map[string]int{}
	var parentIdExceptionList []uint

	count, err := s.db.CountQueuedDescribeResourceJobs()
	if err != nil {
		s.logger.Error("failed to get queue length", zap.String("spot", "CountQueuedDescribeResourceJobs"), zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	if count > MaxQueued {
		return
	}

	drs, err := s.db.ListRandomCreatedDescribeResourceJobs(MaxTriggerPerMinute - int(count))
	if err != nil {
		s.logger.Error("failed to fetch describe resource jobs", zap.String("spot", "ListRandomCreatedDescribeResourceJobs"), zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	for _, dr := range drs {
		ignore := false
		for _, ex := range parentIdExceptionList {
			if dr.ParentJobID == ex {
				ignore = true
			}
		}

		if ignore {
			continue
		}

		ds, err := s.db.GetDescribeSourceJob(dr.ParentJobID)
		if err != nil {
			s.logger.Error("failed to get describe source job", zap.String("spot", "GetDescribeSourceJob"), zap.Error(err), zap.Uint("jobID", dr.ID))
			DescribeJobsCount.WithLabelValues("failure").Inc()
			DescribeResourceJobsCount.WithLabelValues("failure").Inc()
			return
		}

		if accountTriggerCount[ds.SourceID.String()] > MaxTriggerPerAccountPerMinute {
			parentIdExceptionList = append(parentIdExceptionList, dr.ParentJobID)
			continue
		}
		accountTriggerCount[ds.SourceID.String()]++

		s.cloudNativeCallChannel <- CloudNativeCall{
			dr: dr,
			ds: ds,
		}
	}

	DescribeJobsCount.WithLabelValues("successful").Inc()
}

func (s Scheduler) cloudNativeCaller() {
	var c CloudNativeCall
	for {
		c = <-s.cloudNativeCallChannel
		err := s.enqueueCloudNativeDescribeJob(c.dr, c.ds)
		if err != nil {
			s.logger.Error("Failed to enqueueCloudNativeDescribeConnectionJob", zap.Error(err), zap.Uint("jobID", c.dr.ID))
			DescribeJobsCount.WithLabelValues("failure").Inc()
			DescribeResourceJobsCount.WithLabelValues("failure").Inc()
			continue
		}
		DescribeResourceJobsCount.WithLabelValues("successful").Inc()
	}
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

		s.logger.Debug("Source is due for a describe. Creating a job now", zap.String("sourceId", connection.ID.String()))
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
	s.logger.Debug("enqueueCloudNativeDescribeJob",
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

	s.logger.Info("successful job trigger",
		zap.Uint("sourceJobID", ds.ID),
		zap.Uint("jobID", dr.ID),
		zap.String("connectionID", ds.SourceID.String()),
		zap.String("resourceType", dr.ResourceType),
	)

	if err := s.db.UpdateDescribeResourceJobStatus(dr.ID, api.DescribeResourceJobQueued, "", 0); err != nil {
		s.logger.Error("failed to update DescribeResourceJob",
			zap.Uint("sourceJobID", ds.ID),
			zap.Uint("jobID", dr.ID),
			zap.String("connectionID", ds.SourceID.String()),
			zap.String("resourceType", dr.ResourceType),
			zap.Error(err),
		)
	}
	return nil
}
