package describe

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/jq"
	"strings"
	"time"

	complianceAPI "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/kaytu-io/kaytu-engine/pkg/insight"
	insightAPI "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/ticker"
	"go.uber.org/zap"
)

func (s *Scheduler) RunInsightJobScheduler(ctx context.Context) {
	s.logger.Info("Scheduling insight jobs on a timer")

	t := ticker.NewTicker(JobSchedulingInterval, time.Second*10)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleInsightJob(ctx, false)
	}
}

func (s *Scheduler) scheduleInsightJob(ctx context.Context, forceCreate bool) {
	insights, err := s.complianceClient.ListInsightsMetadata(&httpclient.Context{UserRole: api.ViewerRole}, nil)
	if err != nil {
		s.logger.Error("Failed to fetch list of insights", zap.Error(err))
		InsightJobsCount.WithLabelValues("failure").Inc()
		return
	}

	connections, err := s.onboardClient.ListSources(&httpclient.Context{UserRole: api.InternalRole}, nil)
	if err != nil {
		s.logger.Error("Failed to fetch list of sources", zap.Error(err))
		InsightJobsCount.WithLabelValues("failure").Inc()
		return
	}

	if len(connections) == 0 {
		return
	}

	for _, ins := range insights {
		id := fmt.Sprintf("all:%s", strings.ToLower(string(ins.Connector)))
		_, err := s.runInsightJob(ctx, forceCreate, ins, id, id, ins.Connector, nil)
		if err != nil {
			s.logger.Error("Failed to run InsightJob", zap.Error(err))
			InsightJobsCount.WithLabelValues("failure").Inc()
			continue
		}
		InsightJobsCount.WithLabelValues("successful").Inc()
	}
}

func (s *Scheduler) runInsightJob(ctx context.Context, forceCreate bool, ins complianceAPI.Insight, srcID, accountID string, srcType source.Type, resourceCollectionId *string) (uint, error) {
	lastJob, err := s.db.GetLastInsightJobForResourceCollection(ins.ID, srcID, resourceCollectionId)
	if err != nil {
		return 0, err
	}

	if forceCreate || lastJob == nil ||
		lastJob.CreatedAt.Add(s.insightIntervalHours).Before(time.Now()) {

		job := newInsightJob(ins, srcType, srcID, accountID, resourceCollectionId)
		err := s.db.AddInsightJob(&job)
		if err != nil {
			return 0, err
		}

		if err := enqueueInsightJobs(ctx, s.jq, job, ins); err != nil {
			job.Status = insightAPI.InsightJobFailed
			job.FailureMessage = "Failed to enqueue InsightJob"
			s.db.UpdateInsightJobStatus(job)
			return 0, err
		}
		return job.ID, nil
	}
	return 0, nil
}

func enqueueInsightJobs(ctx context.Context, jq *jq.JobQueue, job model.InsightJob, ins complianceAPI.Insight) error {
	bytes, err := json.Marshal(insight.Job{
		JobID:                job.ID,
		InsightID:            job.InsightID,
		SourceID:             job.SourceID,
		AccountID:            job.AccountID,
		SourceType:           ins.Connector,
		Internal:             ins.Internal,
		Query:                ins.Query.QueryToExecute,
		Description:          ins.Description,
		ExecutedAt:           job.CreatedAt.UnixMilli(),
		ResourceCollectionId: job.ResourceCollection,
	})
	if err != nil {
		return err
	}

	if err := jq.Produce(
		ctx,
		insight.JobsQueueName,
		bytes,
		fmt.Sprintf("job-%d", job.ID),
	); err != nil {
		return err
	}
	return nil
}

func newInsightJob(insight complianceAPI.Insight, sourceType source.Type, sourceId, accountId string, resourceCollectionId *string) model.InsightJob {
	return model.InsightJob{
		InsightID:          insight.ID,
		SourceType:         sourceType,
		SourceID:           sourceId,
		AccountID:          accountId,
		Status:             insightAPI.InsightJobCreated,
		FailureMessage:     "",
		ResourceCollection: resourceCollectionId,
	}
}
