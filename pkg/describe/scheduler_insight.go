package describe

import (
	"fmt"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/insight"
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"github.com/kaytu-io/kaytu-util/pkg/source"

	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	insightapi "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"

	complianceapi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"

	"go.uber.org/zap"
)

func (s *Scheduler) RunInsightJobScheduler() {
	s.logger.Info("Scheduling insight jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleInsightJob(false)
	}
}

func (s *Scheduler) scheduleInsightJob(forceCreate bool) {
	insights, err := s.complianceClient.ListInsightsMetadata(&httpclient.Context{UserRole: api2.ViewerRole}, nil)
	if err != nil {
		s.logger.Error("Failed to fetch list of insights", zap.Error(err))
		InsightJobsCount.WithLabelValues("failure").Inc()
		return
	}

	srcs, err := s.onboardClient.ListSources(&httpclient.Context{UserRole: api2.InternalRole}, nil)
	if err != nil {
		s.logger.Error("Failed to fetch list of sources", zap.Error(err))
		InsightJobsCount.WithLabelValues("failure").Inc()
		return
	}

	if len(srcs) == 0 {
		return
	}

	for _, ins := range insights {
		id := fmt.Sprintf("all:%s", strings.ToLower(string(ins.Connector)))
		err := s.runInsightJob(forceCreate, ins, id, id, ins.Connector, nil)
		if err != nil {
			s.logger.Error("Failed to run InsightJob", zap.Error(err))
			InsightJobsCount.WithLabelValues("failure").Inc()
			continue
		}
		InsightJobsCount.WithLabelValues("successful").Inc()
	}

	resourceCollections, err := s.inventoryClient.ListResourceCollections(&httpclient.Context{UserRole: api2.InternalRole})
	if err != nil {
		s.logger.Error("Failed to list resource collections", zap.Error(err))
		return
	}
	for _, resourceCollection := range resourceCollections {
		for _, ins := range insights {
			id := fmt.Sprintf("all:%s", strings.ToLower(string(ins.Connector)))
			err := s.runInsightJob(forceCreate, ins, id, id, ins.Connector, &resourceCollection.ID)
			if err != nil {
				s.logger.Error("Failed to run InsightJob for resourceCollection", zap.Error(err))
				InsightJobsCount.WithLabelValues("failure").Inc()
				continue
			}
			InsightJobsCount.WithLabelValues("successful").Inc()
		}
	}
}

func (s *Scheduler) runInsightJob(forceCreate bool, ins complianceapi.Insight, srcID, accountID string, srcType source.Type, resourceCollectionId *string) error {
	lastJob, err := s.db.GetLastInsightJobForResourceCollection(ins.ID, srcID, resourceCollectionId)
	if err != nil {
		return err
	}

	if forceCreate || lastJob == nil ||
		lastJob.CreatedAt.Add(time.Duration(s.insightIntervalHours)*time.Hour).Before(time.Now()) {

		job := newInsightJob(ins, srcType, srcID, accountID, resourceCollectionId)
		err := s.db.AddInsightJob(&job)
		if err != nil {
			return err
		}

		err = enqueueInsightJobs(s.insightJobQueue, job, ins)
		if err != nil {
			job.Status = insightapi.InsightJobFailed
			job.FailureMessage = "Failed to enqueue InsightJob"
			s.db.UpdateInsightJobStatus(job)
			return err
		}
	}
	return nil
}

func enqueueInsightJobs(q queue.Interface, job InsightJob, ins complianceapi.Insight) error {
	if err := q.Publish(insight.Job{
		JobID:                job.ID,
		InsightID:            job.InsightID,
		SourceID:             job.SourceID,
		AccountID:            job.AccountID,
		SourceType:           ins.Connector,
		Internal:             ins.Internal,
		Query:                ins.Query.QueryToExecute,
		Description:          ins.Description,
		ExecutedAt:           job.CreatedAt.UnixMilli(),
		IsStack:              job.IsStack,
		ResourceCollectionId: job.ResourceCollection,
	}); err != nil {
		return err
	}
	return nil
}

func newInsightJob(insight complianceapi.Insight, sourceType source.Type, sourceId, accountId string, resourceCollectionId *string) InsightJob {
	return InsightJob{
		InsightID:          insight.ID,
		SourceType:         sourceType,
		SourceID:           sourceId,
		AccountID:          accountId,
		Status:             insightapi.InsightJobInProgress,
		FailureMessage:     "",
		IsStack:            false,
		ResourceCollection: resourceCollectionId,
	}
}
