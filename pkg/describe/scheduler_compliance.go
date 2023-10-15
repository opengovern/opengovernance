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
	clientCtx := &httpclient.Context{UserRole: api2.InternalRole}

	benchmarks, err := s.complianceClient.ListBenchmarks(clientCtx)
	if err != nil {
		ComplianceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("error while listing benchmarks", zap.Error(err))
		return fmt.Errorf("error while listing benchmarks: %v", err)
	}

	for _, benchmark := range benchmarks {
		var connections []onboardApi.Connection
		var resourceCollections []string
		assignments, err := s.complianceClient.ListAssignmentsByBenchmark(clientCtx, benchmark.ID)
		if err != nil {
			ComplianceJobsCount.WithLabelValues("failure").Inc()
			s.logger.Error("error while listing assignments", zap.Error(err))
			return fmt.Errorf("error while listing assignments: %v", err)
		}

		for _, assignment := range assignments.Connections {
			if !assignment.Status {
				continue
			}

			connection, err := s.onboardClient.GetSource(clientCtx, assignment.ConnectionID)
			if err != nil {
				ComplianceJobsCount.WithLabelValues("failure").Inc()
				s.logger.Error("error while get source", zap.Error(err))
				return fmt.Errorf("error while get source: %v", err)
			}

			if !connection.IsEnabled() {
				continue
			}

			connections = append(connections, *connection)
		}

		for _, assignment := range assignments.ResourceCollections {
			if !assignment.Status {
				continue
			}
			resourceCollections = append(resourceCollections, assignment.ResourceCollectionID)
		}

		timeAfter := time.Now().Add(time.Duration(-s.complianceIntervalHours) * time.Hour)
		var nullStringPointer *string = nil
		for _, src := range connections {
			connectionID := src.ID.String()
			jobs, err := s.db.ListComplianceReportsWithFilter(
				&timeAfter, nil, &connectionID, nil, &benchmark.ID, &nullStringPointer)
			if err != nil {
				ComplianceJobsCount.WithLabelValues("failure").Inc()
				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
				s.logger.Error("error while listing compliance jobs", zap.Error(err))
				return fmt.Errorf("error while creating compliance job: %v", err)
			}

			if len(jobs) > 0 {
				continue
			}

			crj := newComplianceReportJob(src.ID.String(), src.Connector, benchmark.ID, nil)
			err = s.db.CreateComplianceReportJob(&crj)
			if err != nil {
				ComplianceJobsCount.WithLabelValues("failure").Inc()
				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
				s.logger.Error("error while creating compliance job", zap.Error(err))
				return fmt.Errorf("error while creating compliance job: %v", err)
			}

			enqueueComplianceReportJobs(s.logger, s.db, s.complianceReportJobQueue, &crj)
			ComplianceSourceJobsCount.WithLabelValues("successful").Inc()
		}

		for _, rc := range resourceCollections {
			rc := rc
			rcIDPtr := &rc
			connectionID := "all"
			jobs, err := s.db.ListComplianceReportsWithFilter(
				&timeAfter, nil, &connectionID, nil, &benchmark.ID, &rcIDPtr)
			if err != nil {
				ComplianceJobsCount.WithLabelValues("failure").Inc()
				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
				s.logger.Error("error while listing compliance jobs", zap.Error(err))
				return fmt.Errorf("error while creating compliance job: %v", err)
			}

			if len(jobs) > 0 {
				continue
			}
			if len(benchmark.Connectors) == 0 {
				s.logger.Warn("no connectors found for benchmark - ignoring resource collection", zap.String("benchmark", benchmark.ID), zap.String("resource_collection", rc))
				continue
			}
			crj := newComplianceReportJob(connectionID, benchmark.Connectors[0], benchmark.ID, rcIDPtr)
			err = s.db.CreateComplianceReportJob(&crj)
			if err != nil {
				ComplianceJobsCount.WithLabelValues("failure").Inc()
				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
				s.logger.Error("error while creating compliance job", zap.Error(err))
				return fmt.Errorf("error while creating compliance job: %v", err)
			}

			enqueueComplianceReportJobs(s.logger, s.db, s.complianceReportJobQueue, &crj)
			ComplianceSourceJobsCount.WithLabelValues("successful").Inc()
		}
	}

	ComplianceJobsCount.WithLabelValues("successful").Inc()
	return nil
}

func enqueueComplianceReportJobs(logger *zap.Logger, db Database, q queue.Interface, crj *ComplianceReportJob) {
	nextStatus := complianceapi.ComplianceReportJobInProgress
	errMsg := ""

	nowTime := time.Now().UnixMilli()
	if err := q.Publish(complianceworker.Job{
		JobID:                crj.ID,
		DescribedAt:          nowTime,
		EvaluatedAt:          nowTime,
		ConnectionID:         crj.SourceID,
		BenchmarkID:          crj.BenchmarkID,
		Connector:            crj.SourceType,
		IsStack:              crj.IsStack,
		ResourceCollectionId: crj.ResourceCollection,
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
