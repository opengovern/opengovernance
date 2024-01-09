package compliance

import (
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
) ([]*model.ComplianceRunner, error) {
	ctx := &httpclient.Context{UserRole: api2.InternalRole}
	var runners []*model.ComplianceRunner

	benchmark, err := s.complianceClient.GetBenchmark(ctx, benchmarkID)
	if err != nil {
		return nil, err
	}

	for _, child := range benchmark.Children {
		childRunners, err := s.buildRunners(parentJobID, connectionID, resourceCollectionID, rootBenchmarkID, append(parentBenchmarkIDs, benchmarkID), child)
		if err != nil {
			return nil, err
		}

		runners = append(runners, childRunners...)
	}

	for _, controlID := range benchmark.Controls {
		control, err := s.complianceClient.GetControl(ctx, controlID)
		if err != nil {
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
				return nil, err
			}

			cv, err := v.GetCallers()
			if err != nil {
				return nil, err
			}

			cv = append(cv, cr...)
			err = v.SetCallers(cv)
			if err != nil {
				return nil, err
			}
		} else {
			v = r
		}
		uniqueMap[r.QueryID] = v
	}

	var jobs []*model.ComplianceRunner
	for _, v := range uniqueMap {
		jobs = append(jobs, v)
	}
	return jobs, nil
}

func (s *JobScheduler) CreateComplianceReportJobs(benchmarkID string) (uint, error) {
	assignments, err := s.complianceClient.ListAssignmentsByBenchmark(&httpclient.Context{UserRole: api2.InternalRole}, benchmarkID)
	if err != nil {
		return 0, err
	}

	job := model.ComplianceJob{
		BenchmarkID: benchmarkID,
		Status:      model.ComplianceJobCreated,
		IsStack:     false,
	}
	err = s.db.CreateComplianceJob(&job)
	if err != nil {
		return 0, err
	}

	// delete old runners
	err = s.db.DeleteOldRunnerJob(&job.ID)
	if err != nil {
		s.logger.Error("error while deleting old runners", zap.Error(err))
		return 0, err
	}

	var allRunners []*model.ComplianceRunner
	for _, it := range assignments.Connections {
		connection := it
		runners, err := s.buildRunners(job.ID, &connection.ConnectionID, nil, benchmarkID, nil, benchmarkID)
		if err != nil {
			return 0, err
		}
		allRunners = append(allRunners, runners...)
	}

	// We don't need to create runners for resource collections anymore because we are handling it in the summarizer
	//for _, it := range assignments.ResourceCollections {
	//	resourceCollection := it
	//	runners, err := s.buildRunners(job.ID, nil, &resourceCollection.ResourceCollectionID, benchmarkID, nil, benchmarkID)
	//	if err != nil {
	//		return 0, err
	//	}
	//	allRunners = append(allRunners, runners...)
	//}

	err = s.db.CreateRunnerJobs(allRunners)
	if err != nil {
		return 0, err
	}

	return job.ID, nil
}
