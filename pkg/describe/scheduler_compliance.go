package describe

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	kafka2 "github.com/kaytu-io/kaytu-util/pkg/kafka"
	"gorm.io/gorm"
	"time"

	complianceworker "github.com/kaytu-io/kaytu-engine/pkg/compliance/worker"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
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
			ComplianceJobsCount.WithLabelValues("failure").Inc()
			continue
		}
	}
}

func (s *Scheduler) scheduleComplianceJob() error {
	s.logger.Info("scheduleComplianceJob")
	clientCtx := &httpclient.Context{UserRole: api2.InternalRole}

	benchmarks, err := s.complianceClient.ListBenchmarks(clientCtx)
	if err != nil {
		return fmt.Errorf("error while listing benchmarks: %v", err)
	}

	for _, benchmark := range benchmarks {
		var connections []onboardApi.Connection
		var resourceCollections []string
		assignments, err := s.complianceClient.ListAssignmentsByBenchmark(clientCtx, benchmark.ID)
		if err != nil {
			return fmt.Errorf("error while listing assignments: %v", err)
		}

		for _, assignment := range assignments.Connections {
			if !assignment.Status {
				continue
			}

			connection, err := s.onboardClient.GetSource(clientCtx, assignment.ConnectionID)
			if err != nil {
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

		if len(connections) == 0 && len(resourceCollections) == 0 {
			continue
		}

		complianceJob, err := s.db.GetLastComplianceJob(benchmark.ID)
		if err != nil {
			return err
		}

		timeAt := time.Now().Add(time.Duration(-s.complianceIntervalHours) * time.Hour)
		if complianceJob == nil ||
			complianceJob.CreatedAt.Before(timeAt) {
			_, err := s.triggerComplianceReportJobs(benchmark.ID)
			if err != nil {
				return err
			}

			ComplianceJobsCount.WithLabelValues("successful").Inc()
		}
	}

	return nil
}

func newComplianceReportJob(benchmarkID string) model.ComplianceJob {
	return model.ComplianceJob{
		Model:          gorm.Model{},
		BenchmarkID:    benchmarkID,
		Status:         api.ComplianceReportJobCreated,
		FailureMessage: "",
		IsStack:        false,
	}
}

func (s *Scheduler) triggerComplianceReportJobs(benchmarkID string) (uint, error) {
	jobModel := newComplianceReportJob(benchmarkID)
	err := s.db.CreateComplianceJob(&jobModel)
	if err != nil {
		return 0, err
	}

	job := complianceworker.Job{
		ID:          jobModel.ID,
		CreatedAt:   jobModel.CreatedAt,
		BenchmarkID: benchmarkID,
		IsStack:     false,
	}
	jobJson, err := json.Marshal(job)
	if err != nil {
		_ = s.db.UpdateComplianceJob(job.ID, api.ComplianceReportJobCompletedWithFailure, err.Error())
		return 0, err
	}

	msg := kafka2.Msg(fmt.Sprintf("job-%d", job.ID), jobJson, "", complianceworker.JobQueue, kafka.PartitionAny)
	_, err = kafka2.SyncSend(s.logger, s.kafkaProducer, []*kafka.Message{msg}, nil)
	if err != nil {
		_ = s.db.UpdateComplianceJob(job.ID, api.ComplianceReportJobCompletedWithFailure, err.Error())
		return 0, err
	}

	_ = s.db.UpdateComplianceJob(job.ID, api.ComplianceReportJobInProgress, "")
	return job.ID, nil
}

func (s *Scheduler) RunComplianceReportJobResultsConsumer() error {
	ctx := context.Background()
	consumer, err := kafka2.NewTopicConsumer(ctx, s.kafkaServers, complianceworker.ResultQueue, complianceworker.ConsumerGroup)
	if err != nil {
		return err
	}
	msgs := consumer.Consume(ctx, s.logger)
	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg := <-msgs:
			var result complianceworker.JobResult
			if err := json.Unmarshal(msg.Value, &result); err != nil {
				s.logger.Error("Failed to unmarshal ComplianceReportJob results", zap.Error(err))

				err := consumer.Commit(msg)
				if err != nil {
					s.logger.Error("Failed committing message", zap.Error(err))
				}
				continue
			}

			s.logger.Info("Processing ReportJobResult for Job",
				zap.Uint("jobId", result.Job.ID),
				zap.String("status", string(result.Status)),
			)
			err := s.db.UpdateComplianceJob(result.Job.ID, result.Status, result.Error)
			if err != nil {
				s.logger.Error("Failed to update the status of ComplianceReportJob",
					zap.Uint("jobId", result.Job.ID),
					zap.Error(err))

				err := consumer.Commit(msg)
				if err != nil {
					s.logger.Error("Failed committing message", zap.Error(err))
				}
				continue
			}

			err = consumer.Commit(msg)
			if err != nil {
				s.logger.Error("Failed committing message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateComplianceJobsTimedOut(s.complianceTimeoutHours)
			if err != nil {
				s.logger.Error("Failed to update timed out ComplianceReportJob", zap.Error(err))
			}
		}
	}
}
