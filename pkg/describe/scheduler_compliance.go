package describe

import (
	"fmt"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-util/pkg/queue"

	complianceapi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	complianceworker "github.com/kaytu-io/kaytu-engine/pkg/compliance/worker"
	"go.uber.org/zap"

	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
)

func (s *Scheduler) RunComplianceJobScheduler() {
	s.logger.Info("Scheduling compliance jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		err := s.scheduleComplianceJob()
		if err != nil {
			s.logger.Error("failed to run scheduleComplianceJob", zap.Error(err))
			continue
		}
	}
}

func (s *Scheduler) scheduleComplianceJob() error {
	s.logger.Info("scheduleComplianceJob")

	sources, err := s.onboardClient.ListSources(&httpclient.Context{UserRole: api2.KaytuAdminRole}, nil)
	if err != nil {
		ComplianceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("error while listing sources", zap.Error(err))
		return fmt.Errorf("error while listing sources: %v", err)
	}

	for _, src := range sources {
		if !src.IsEnabled() {
			continue
		}
		ctx := &httpclient.Context{
			UserRole: api2.ViewerRole,
		}
		benchmarks, err := s.complianceClient.GetAllBenchmarkAssignmentsBySourceId(ctx, src.ID.String())
		if err != nil {
			ComplianceJobsCount.WithLabelValues("failure").Inc()
			s.logger.Error("error while getting benchmark assignments", zap.Error(err))
			return fmt.Errorf("error while getting benchmark assignments: %v", err)
		}

		for _, b := range benchmarks {
			timeAfter := time.Now().Add(time.Duration(-s.complianceIntervalHours) * time.Hour)
			jobs, err := s.db.ListComplianceReportsWithFilter(&timeAfter, nil, &b.ConnectionId, nil, &b.BenchmarkId)
			if err != nil {
				ComplianceJobsCount.WithLabelValues("failure").Inc()
				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
				s.logger.Error("error while listing compliance jobs", zap.Error(err))
				return fmt.Errorf("error while creating compliance job: %v", err)
			}

			if len(jobs) > 0 {
				continue
			}

			crj := newComplianceReportJob(src.ID.String(), src.Connector, b.BenchmarkId)
			err = s.db.CreateComplianceReportJob(&crj)
			if err != nil {
				ComplianceJobsCount.WithLabelValues("failure").Inc()
				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
				s.logger.Error("error while creating compliance job", zap.Error(err))
				return fmt.Errorf("error while creating compliance job: %v", err)
			}

			enqueueComplianceReportJobs(s.logger, s.db, s.complianceReportJobQueue, src, &crj)

			ComplianceSourceJobsCount.WithLabelValues("successful").Inc()
		}
	}
	ComplianceJobsCount.WithLabelValues("successful").Inc()
	return nil

}

func enqueueComplianceReportJobs(logger *zap.Logger, db Database, q queue.Interface, a onboardApi.Connection, crj *ComplianceReportJob) {
	nextStatus := complianceapi.ComplianceReportJobInProgress
	errMsg := ""

	nowTime := time.Now().UnixMilli()
	if err := q.Publish(complianceworker.Job{
		JobID:        crj.ID,
		DescribedAt:  nowTime,
		EvaluatedAt:  nowTime,
		ConnectionID: crj.SourceID,
		BenchmarkID:  crj.BenchmarkID,
		ConfigReg:    a.Credential.Config.(string),
		Connector:    a.Connector,
		IsStack:      crj.IsStack,
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
