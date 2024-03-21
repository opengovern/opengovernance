package compliance

import (
	complianceApi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"time"

	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"go.uber.org/zap"
)

func (s *JobScheduler) buildRunners(
	parentJobID uint,
	connectionID *string,
	resourceCollectionID *string,
	rootBenchmarkID string,
	parentBenchmarkIDs []string,
	benchmarkID string,
	currentRunnerExistMap map[string]bool,
) ([]*model.ComplianceRunner, error) {
	ctx := &httpclient.Context{UserRole: api2.InternalRole}
	var runners []*model.ComplianceRunner

	benchmark, err := s.complianceClient.GetBenchmark(ctx, benchmarkID)
	if err != nil {
		s.logger.Error("error while getting benchmark", zap.Error(err), zap.String("benchmarkID", benchmarkID))
		return nil, err
	}
	if currentRunnerExistMap == nil {
		currentRunners, err := s.db.GetRunnersByParentJobID(parentJobID)
		if err != nil {
			s.logger.Error("error while getting current runners", zap.Error(err))
			return nil, err
		}
		currentRunnerExistMap = make(map[string]bool)
		for _, r := range currentRunners {
			currentRunnerExistMap[r.GetKeyIdentifier()] = true
		}
	}

	for _, child := range benchmark.Children {
		childRunners, err := s.buildRunners(parentJobID, connectionID, resourceCollectionID, rootBenchmarkID, append(parentBenchmarkIDs, benchmarkID), child, currentRunnerExistMap)
		if err != nil {
			s.logger.Error("error while building child runners", zap.Error(err))
			return nil, err
		}

		runners = append(runners, childRunners...)
	}

	for _, controlID := range benchmark.Controls {
		control, err := s.complianceClient.GetControl(ctx, controlID)
		if err != nil {
			s.logger.Error("error while getting control", zap.Error(err), zap.String("controlID", controlID))
			return nil, err
		}

		if control.Query == nil {
			continue
		}

		callers := runner.Caller{
			RootBenchmark:      rootBenchmarkID,
			ParentBenchmarkIDs: append(parentBenchmarkIDs, benchmarkID),
			ControlID:          control.ID,
			ControlSeverity:    control.Severity,
		}

		runnerJob := model.ComplianceRunner{
			BenchmarkID:          rootBenchmarkID,
			QueryID:              control.Query.ID,
			ConnectionID:         connectionID,
			ResourceCollectionID: resourceCollectionID,
			ParentJobID:          parentJobID,
			StartedAt:            time.Time{},
			RetryCount:           0,
			Status:               runner.ComplianceRunnerCreated,
			FailureMessage:       "",
		}
		err = runnerJob.SetCallers([]runner.Caller{callers})
		if err != nil {
			return nil, err
		}
		runners = append(runners, &runnerJob)
	}

	uniqueMap := map[string]*model.ComplianceRunner{}
	for _, r := range runners {
		v, ok := uniqueMap[r.QueryID]
		if ok {
			cr, err := r.GetCallers()
			if err != nil {
				s.logger.Error("error while getting callers", zap.Error(err))
				return nil, err
			}

			cv, err := v.GetCallers()
			if err != nil {
				s.logger.Error("error while getting callers", zap.Error(err))
				return nil, err
			}

			cv = append(cv, cr...)
			err = v.SetCallers(cv)
			if err != nil {
				s.logger.Error("error while setting callers", zap.Error(err))
				return nil, err
			}
		} else {
			v = r
		}
		uniqueMap[r.QueryID] = v
	}

	var jobs []*model.ComplianceRunner
	for _, v := range uniqueMap {
		if !currentRunnerExistMap[v.GetKeyIdentifier()] {
			jobs = append(jobs, v)
		}
	}
	return jobs, nil
}

func (s *JobScheduler) CreateComplianceReportJobs(benchmarkID string,
	lastJob *model.ComplianceJob, connectionIDs []string) (uint, error) {
	// delete old runners
	if lastJob != nil {
		err := s.db.DeleteOldRunnerJob(&lastJob.ID)
		if err != nil {
			s.logger.Error("error while deleting old runners", zap.Error(err))
			return 0, err
		}
	} else {
		err := s.db.DeleteOldRunnerJob(nil)
		if err != nil {
			s.logger.Error("error while deleting old runners", zap.Error(err))
			return 0, err
		}
	}

	job := model.ComplianceJob{
		BenchmarkID:         benchmarkID,
		Status:              model.ComplianceJobCreated,
		AreAllRunnersQueued: false,
		ConnectionIDs:       connectionIDs,
		IsStack:             false,
	}
	err := s.db.CreateComplianceJob(nil, &job)
	if err != nil {
		s.logger.Error("error while creating compliance job", zap.Error(err))
		return 0, err
	}

	return job.ID, nil
}

func (s *JobScheduler) enqueueRunnersCycle() error {
	s.logger.Info("enqueue runners cycle started")
	var err error
	jobsWithUnqueuedRunners, err := s.db.ListComplianceJobsWithUnqueuedRunners()
	if err != nil {
		s.logger.Error("error while listing jobs with unqueued runners", zap.Error(err))
		return err
	}
	s.logger.Info("jobs with unqueued runners", zap.Int("count", len(jobsWithUnqueuedRunners)))
	for _, job := range jobsWithUnqueuedRunners {
		s.logger.Info("processing job with unqueued runners", zap.Uint("jobID", job.ID))
		var allRunners []*model.ComplianceRunner
		var assignments *complianceApi.BenchmarkAssignedEntities
		if len(job.ConnectionIDs) > 0 {
			connections, err := s.onboardClient.GetSources(&httpclient.Context{UserRole: api2.InternalRole}, job.ConnectionIDs)
			if err != nil {
				s.logger.Error("error while getting sources", zap.Error(err))
				continue
			}
			assignments = &complianceApi.BenchmarkAssignedEntities{}
			for _, connection := range connections {
				assignment := complianceApi.BenchmarkAssignedConnection{
					ConnectionID:           connection.ID.String(),
					ProviderConnectionID:   connection.ConnectionID,
					ProviderConnectionName: connection.ConnectionName,
					Connector:              connection.Connector,
					Status:                 true,
				}
				assignments.Connections = append(assignments.Connections, assignment)
			}
		} else {
			assignments, err = s.complianceClient.ListAssignmentsByBenchmark(&httpclient.Context{UserRole: api2.InternalRole}, job.BenchmarkID)
			if err != nil {
				s.logger.Error("error while listing assignments", zap.Error(err))
				continue
			}
		}
		for _, it := range assignments.Connections {
			if !it.Status {
				continue
			}
			connection := it
			runners, err := s.buildRunners(job.ID, &connection.ConnectionID, nil, job.BenchmarkID, nil, job.BenchmarkID, nil)
			if err != nil {
				s.logger.Error("error while building runners", zap.Error(err))
				return err
			}
			allRunners = append(allRunners, runners...)
		}
		if len(allRunners) > 0 {
			s.logger.Info("creating runners", zap.Int("count", len(allRunners)), zap.Uint("jobID", job.ID))
			err = s.db.CreateRunnerJobs(nil, allRunners)
			if err != nil {
				s.logger.Error("error while creating runners", zap.Error(err))
				return err
			}
		} else {
			s.logger.Info("no runners to create", zap.Uint("jobID", job.ID))
			err = s.db.UpdateComplianceJobAreAllRunnersQueued(job.ID, true)
			if err != nil {
				s.logger.Error("error while updating compliance job", zap.Error(err))
				return err
			}
		}
	}

	return nil
}
