package describe

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	apiDescribe "github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/enums"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	apiOnboard "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-util/pkg/concurrency"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
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
				err := s.enqueueCloudNativeDescribeJob(dr, ds, cred.Secret, s.WorkspaceName, ("stack-" + ds.SourceID.String()))
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
			s.logger.Error("failed to describe connection", zap.String("connection_id", connection.ID.String()), zap.Error(err))
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
		}, connection.ID.String())
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
		s.logger.Debug("Source is due for a describe. Creating a job now", zap.String("sourceId", connection.ID.String()))

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
			}, connection.ID.String(), apiOnboard.ConnectionLifecycleStateOnboard)
			if err != nil {
				s.logger.Warn("Failed to set connection lifecycle state", zap.String("connection_id", connection.ID.String()), zap.Error(err))
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
		zap.String("connectionID", ds.SourceID.String()),
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
			SourceID:     ds.SourceID.String(),
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
		zap.String("connectionID", ds.SourceID.String()),
		zap.String("resourceType", dr.ResourceType),
	)

	if err := s.db.QueueDescribeResourceJob(dr.ID); err != nil {
		s.logger.Error("failed to QueueDescribeResourceJob",
			zap.Uint("sourceJobID", ds.ID),
			zap.Uint("jobID", dr.ID),
			zap.String("connectionID", ds.SourceID.String()),
			zap.String("resourceType", dr.ResourceType),
			zap.Error(err),
		)
	}
	return nil
}
