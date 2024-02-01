package compliance

import (
	"fmt"
	"go.uber.org/zap"
	"time"

	authAPI "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	onboardAPI "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
)

func (s *JobScheduler) runScheduler() error {
	s.logger.Info("scheduleComplianceJob")
	clientCtx := &httpclient.Context{UserRole: authAPI.InternalRole}

	benchmarks, err := s.complianceClient.ListBenchmarks(clientCtx)
	if err != nil {
		s.logger.Error("error while listing benchmarks", zap.Error(err))
		return fmt.Errorf("error while listing benchmarks: %v", err)
	}

	allConnections, err := s.onboardClient.ListSources(clientCtx, nil)
	if err != nil {
		s.logger.Error("error while listing allConnections", zap.Error(err))
		return fmt.Errorf("error while listing allConnections: %v", err)
	}
	connectionsMap := make(map[string]*onboardAPI.Connection)
	for _, connection := range allConnections {
		connection := connection
		connectionsMap[connection.ID.String()] = &connection
	}

	for _, benchmark := range benchmarks {
		var connections []onboardAPI.Connection
		assignments, err := s.complianceClient.ListAssignmentsByBenchmark(clientCtx, benchmark.ID)
		if err != nil {
			s.logger.Error("error while listing assignments", zap.Error(err))
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
			s.logger.Error("error while getting last compliance job", zap.Error(err))
			return err
		}

		timeAt := time.Now().Add(-s.complianceIntervalHours)
		if complianceJob == nil ||
			complianceJob.CreatedAt.Before(timeAt) {

			_, err := s.CreateComplianceReportJobs(benchmark.ID)
			if err != nil {
				s.logger.Error("error while creating compliance job", zap.Error(err))
				return err
			}

			ComplianceJobsCount.WithLabelValues("successful").Inc()
		}
	}

	return nil
}
