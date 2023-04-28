package describe

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"go.uber.org/zap"
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
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		} else {
			DescribeSourceJobsCount.WithLabelValues("successful").Inc()
		}
	}

	drs, err := s.db.ListCreatedDescribeResourceJobs()
	if err != nil {
		s.logger.Error("Failed to fetch all describe source jobs", zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	for _, ds := range drs {
		err = s.enqueueCloudNativeDescribeJob(ds)
		if err != nil {
			s.logger.Error("Failed to enqueueCloudNativeDescribeConnectionJob", zap.Error(err), zap.Uint("jobID", ds.ID))
			DescribeJobsCount.WithLabelValues("failure").Inc()
			return
		}
	}

	DescribeJobsCount.WithLabelValues("successful").Inc()
}

func (s Scheduler) describeConnection(connection Source, scheduled bool) error {
	job, err := s.db.GetLastDescribeSourceJob(connection.ID)
	if err != nil {
		return err
	}

	if !scheduled || // manual
		job == nil || job.UpdatedAt.Before(time.Now().Add(time.Duration(-s.describeIntervalHours)*time.Hour)) {

		healthCheckedSrc, err := s.onboardClient.GetSourceHealthcheck(&httpclient.Context{
			UserRole: api2.EditorRole,
		}, connection.ID.String())
		if err != nil {
			return err
		}

		if scheduled && healthCheckedSrc.AssetDiscoveryMethod != source.AssetDiscoveryMethodTypeScheduled {
			return errors.New("asset discovery is not scheduled")
		}

		if healthCheckedSrc.HealthState == source.HealthStatusUnhealthy {
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
			return err
		}
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
	case api.SourceCloudAWS:
		resourceTypes = aws.ListResourceTypes()
	case api.SourceCloudAzure:
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

func (s Scheduler) enqueueCloudNativeDescribeJob(dr DescribeResourceJob) error {
	ds, err := s.db.GetDescribeSourceJob(dr.ParentJobID)
	if err != nil {
		return err
	}

	s.logger.Info("enqueueCloudNativeDescribeJob",
		zap.Uint("sourceJobID", ds.ID),
		zap.Uint("jobID", dr.ID),
		zap.String("connectionID", ds.SourceID.String()),
		zap.String("resourceType", dr.ResourceType),
	)

	if ds.Status == api.DescribeSourceJobCreated {
		err = s.db.UpdateDescribeSourceJob(ds.ID, api.DescribeSourceJobInProgress)
		if err != nil {
			return err
		}
	}

	src, err := s.db.GetSourceByUUID(ds.SourceID)
	if err != nil {
		return err
	}

	input := LambdaDescribeWorkerInput{
		WorkspaceId:      CurrentWorkspaceID,
		DescribeEndpoint: s.describeEndpoint,
		KeyARN:           s.keyARN,
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
	req, err := http.NewRequest(http.MethodPost, LambdaFuncURL, bytes.NewBuffer(lambdaRequest))
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

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("failed to trigger cloud native worker due to %d: %s", resp.StatusCode, string(resBody))
	}

	s.logger.Info("Successful job trigger",
		zap.Uint("sourceJobID", ds.ID),
		zap.Uint("jobID", dr.ID),
		zap.String("connectionID", ds.SourceID.String()),
		zap.String("resourceType", dr.ResourceType),
	)

	if err := s.db.UpdateDescribeResourceJobStatus(dr.ID, api.DescribeResourceJobQueued, fmt.Sprintf("%v", err)); err != nil {
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
