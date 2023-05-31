package describe

import (
	"fmt"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/insight"

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	insightapi "gitlab.com/keibiengine/keibi-engine/pkg/insight/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"

	complianceapi "gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"

	"go.uber.org/zap"
)

func (s Scheduler) RunInsightJobScheduler() {
	s.logger.Info("Scheduling insight jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleInsightJob(false)
	}
}

func (s Scheduler) scheduleInsightJob(forceCreate bool) {
	srcs, err := s.db.ListSources()
	if err != nil {
		s.logger.Error("Failed to fetch list of sources", zap.Error(err))
		InsightJobsCount.WithLabelValues("failure").Inc()
		return
	}

	insights, err := s.complianceClient.ListInsightsMetadata(&httpclient.Context{UserRole: api2.ViewerRole}, nil)
	if err != nil {
		s.logger.Error("Failed to fetch list of insights", zap.Error(err))
		InsightJobsCount.WithLabelValues("failure").Inc()
		return
	}

	for _, ins := range insights {
		for _, src := range srcs {
			if ins.Connector != source.Nil && src.Type != ins.Connector {
				// insight is not for this source
				continue
			}

			err := s.runInsightJob(forceCreate, ins, src.ID.String(), src.AccountID, src.Type)
			if err != nil {
				s.logger.Error("Failed to run InsightJob", zap.Error(err))
				InsightJobsCount.WithLabelValues("failure").Inc()
				continue
			}
			InsightJobsCount.WithLabelValues("successful").Inc()
		}

		id := fmt.Sprintf("all:%s", strings.ToLower(string(ins.Connector)))
		err := s.runInsightJob(forceCreate, ins, id, id, ins.Connector)
		if err != nil {
			s.logger.Error("Failed to run InsightJob", zap.Error(err))
			InsightJobsCount.WithLabelValues("failure").Inc()
			continue
		}
		InsightJobsCount.WithLabelValues("successful").Inc()
	}
}

func (s *Scheduler) runInsightJob(forceCreate bool, ins complianceapi.Insight, srcID, accountID string, srcType source.Type) error {
	lastJob, err := s.db.GetLastInsightJob(ins.ID, srcID)
	if err != nil {
		return err
	}

	if forceCreate || lastJob == nil ||
		lastJob.CreatedAt.Add(time.Duration(s.insightIntervalHours)*time.Hour).Before(time.Now()) {

		job := newInsightJob(ins, string(srcType), srcID, accountID, "")
		err := s.db.AddInsightJob(&job)
		if err != nil {
			return err
		}

		err = enqueueInsightJobs(s.db, s.insightJobQueue, job, ins)
		if err != nil {
			job.Status = insightapi.InsightJobFailed
			job.FailureMessage = "Failed to enqueue InsightJob"
			s.db.UpdateInsightJobStatus(job)
			return err
		}
	}
	return nil
}

func enqueueInsightJobs(db Database, q queue.Interface, job InsightJob, ins complianceapi.Insight) error {
	var lastDayJobID, lastWeekJobID, lastMonthJobID, lastQuarterJobID, lastYearJobID uint

	lastDay, err := db.GetOldCompletedInsightJob(job.InsightID, 1)
	if err != nil {
		return err
	}
	if lastDay != nil {
		lastDayJobID = lastDay.ID
	}

	lastWeek, err := db.GetOldCompletedInsightJob(job.InsightID, 7)
	if err != nil {
		return err
	}
	if lastWeek != nil {
		lastWeekJobID = lastWeek.ID
	}

	lastMonth, err := db.GetOldCompletedInsightJob(job.InsightID, 30)
	if err != nil {
		return err
	}
	if lastMonth != nil {
		lastMonthJobID = lastMonth.ID
	}

	lastQuarter, err := db.GetOldCompletedInsightJob(job.InsightID, 93)
	if err != nil {
		return err
	}
	if lastQuarter != nil {
		lastQuarterJobID = lastQuarter.ID
	}

	lastYear, err := db.GetOldCompletedInsightJob(job.InsightID, 428)
	if err != nil {
		return err
	}
	if lastYear != nil {
		lastYearJobID = lastYear.ID
	}

	if err := q.Publish(insight.Job{
		JobID:            job.ID,
		QueryID:          job.InsightID,
		SourceID:         job.SourceID,
		ScheduleJobUUID:  job.ScheduleUUID,
		AccountID:        job.AccountID,
		SourceType:       ins.Connector,
		Internal:         ins.Internal,
		Query:            ins.Query.QueryToExecute,
		Description:      ins.Description,
		ExecutedAt:       job.CreatedAt.UnixMilli(),
		LastDayJobID:     lastDayJobID,
		LastWeekJobID:    lastWeekJobID,
		LastMonthJobID:   lastMonthJobID,
		LastQuarterJobID: lastQuarterJobID,
		LastYearJobID:    lastYearJobID,
	}); err != nil {
		return err
	}
	return nil
}

func newInsightJob(insight complianceapi.Insight, sourceType, sourceId, accountId string, scheduleUUID string) InsightJob {
	srcType, _ := source.ParseType(sourceType)
	return InsightJob{
		InsightID:      insight.ID,
		SourceID:       sourceId,
		AccountID:      accountId,
		ScheduleUUID:   scheduleUUID,
		SourceType:     srcType,
		Status:         insightapi.InsightJobInProgress,
		FailureMessage: "",
	}
}
