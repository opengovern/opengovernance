package compliance

import (
	"fmt"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"time"
)

func (s *JobScheduler) runScheduler() error {
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

		timeAt := time.Now().Add(time.Duration(-s.conf.ComplianceIntervalHours) * time.Hour)
		if complianceJob == nil ||
			complianceJob.CreatedAt.Before(timeAt) {

			_, err := s.CreateComplianceReportJobs(benchmark.ID, complianceJob)
			if err != nil {
				return err
			}

			ComplianceJobsCount.WithLabelValues("successful").Inc()
		}
	}

	return nil
}
