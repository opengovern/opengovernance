package compliance

import (
	"fmt"
	"time"

	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
)

func (s *JobScheduler) runScheduler() error {
	s.logger.Info("scheduleComplianceJob")
	clientCtx := &httpclient.Context{UserRole: api2.InternalRole}

	benchmarks, err := s.complianceClient.ListBenchmarks(clientCtx)
	if err != nil {
		return fmt.Errorf("error while listing benchmarks: %v", err)
	}

	allConnections, err := s.onboardClient.ListSources(clientCtx, nil)
	if err != nil {
		return fmt.Errorf("error while listing allConnections: %v", err)
	}
	connectionsMap := make(map[string]*onboardApi.Connection)
	for _, connection := range allConnections {
		connection := connection
		connectionsMap[connection.ID.String()] = &connection
	}

	for _, benchmark := range benchmarks {
		var connections []onboardApi.Connection
		assignments, err := s.complianceClient.ListAssignmentsByBenchmark(clientCtx, benchmark.ID)
		if err != nil {
			return fmt.Errorf("error while listing assignments: %v", err)
		}

		for _, assignment := range assignments.Connections {
			if !assignment.Status {
				continue
			}

			connection := connectionsMap[assignment.ConnectionID]

			if !connection.IsEnabled() {
				continue
			}

			connections = append(connections, *connection)
		}

		if len(connections) == 0 {
			continue
		}

		complianceJob, err := s.db.GetLastComplianceJob(benchmark.ID)
		if err != nil {
			return err
		}

		timeAt := time.Now().Add(-s.complianceIntervalHours)
		if complianceJob == nil ||
			complianceJob.CreatedAt.Before(timeAt) {

			_, err := s.CreateComplianceReportJobs(benchmark.ID)
			if err != nil {
				return err
			}

			ComplianceJobsCount.WithLabelValues("successful").Inc()
		}
	}

	return nil
}
