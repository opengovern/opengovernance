package describe

import (
	"fmt"
	complianceapi "gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	complianceworker "gitlab.com/keibiengine/keibi-engine/pkg/compliance/worker"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/queue"
	"go.uber.org/zap"
	"time"

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

func (s Scheduler) RunComplianceJobScheduler() {
	s.logger.Info("Scheduling compliance jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleComplianceJob()
	}
}

func (s Scheduler) scheduleComplianceJob() error {
	s.logger.Info("scheduleComplianceJob")

	scheduleJob, err := s.db.FetchLastScheduleJob() //TODO-Saleh remove schedule job
	if err != nil {
		ComplianceJobsCount.WithLabelValues("failure").Inc()
		return fmt.Errorf("error while getting last schedule job: %v", err)
	}

	sources, err := s.db.ListSources()
	if err != nil {
		ComplianceJobsCount.WithLabelValues("failure").Inc()
		return fmt.Errorf("error while listing sources: %v", err)
	}

	for _, src := range sources {
		ctx := &httpclient.Context{
			UserRole: api2.ViewerRole,
		}
		benchmarks, err := s.complianceClient.GetAllBenchmarkAssignmentsBySourceId(ctx, src.ID)
		if err != nil {
			ComplianceJobsCount.WithLabelValues("failure").Inc()
			return fmt.Errorf("error while getting benchmark assignments: %v", err)
		}

		for _, b := range benchmarks {
			crj := newComplianceReportJob(src.ID.String(), source.Type(src.Type), b.BenchmarkId, scheduleJob.ID)
			err := s.db.CreateComplianceReportJob(&crj)
			if err != nil {
				ComplianceJobsCount.WithLabelValues("failure").Inc()
				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
				return fmt.Errorf("error while creating compliance job: %v", err)
			}

			enqueueComplianceReportJobs(s.logger, s.db, s.complianceReportJobQueue, src, &crj, scheduleJob)

			err = s.db.UpdateSourceReportGenerated(src.ID.String(), s.complianceIntervalHours)
			if err != nil {
				ComplianceJobsCount.WithLabelValues("failure").Inc()
				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
				return fmt.Errorf("error while updating compliance job: %v", err)
			}
			ComplianceSourceJobsCount.WithLabelValues("successful").Inc()
		}
	}
	ComplianceJobsCount.WithLabelValues("successful").Inc()
	return nil

}

func enqueueComplianceReportJobs(logger *zap.Logger, db Database, q queue.Interface,
	a Source, crj *ComplianceReportJob, scheduleJob *ScheduleJob) {
	nextStatus := complianceapi.ComplianceReportJobInProgress
	errMsg := ""

	if err := q.Publish(complianceworker.Job{
		JobID:         crj.ID,
		ScheduleJobID: scheduleJob.ID,
		DescribedAt:   scheduleJob.CreatedAt.UnixMilli(),
		EvaluatedAt:   time.Now().UnixMilli(),
		ConnectionID:  crj.SourceID,
		BenchmarkID:   crj.BenchmarkID,
		ConfigReg:     a.ConfigRef,
		Connector:     source.Type(a.Type),
	}); err != nil {
		logger.Error("Failed to queue ComplianceReportJob",
			zap.Uint("jobId", crj.ID),
			zap.Error(err),
		)

		nextStatus = complianceapi.ComplianceReportJobCompletedWithFailure
		errMsg = fmt.Sprintf("queue: %s", err.Error())
	}

	if err := db.UpdateComplianceReportJob(crj.ID, nextStatus, 0, errMsg); err != nil {
		logger.Error("Failed to update ComplianceReportJob",
			zap.Uint("jobId", crj.ID),
			zap.Error(err),
		)
	}

	crj.Status = nextStatus
}
