package compliance

import (
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"time"
)

func (s *JobScheduler) buildRunners(
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
		childRunners, err := s.buildRunners(connectionID, resourceCollectionID, rootBenchmarkID, append(parentBenchmarkIDs, benchmarkID), child)
		if err != nil {
			return nil, err
		}

		runners = append(runners, childRunners...)
	}

	for _, policyID := range benchmark.Policies {
		policy, err := s.complianceClient.GetPolicy(ctx, policyID)
		if err != nil {
			return nil, err
		}

		if policy.QueryID == nil {
			continue
		}

		callers := runner.Caller{
			RootBenchmark:      rootBenchmarkID,
			ParentBenchmarkIDs: append(parentBenchmarkIDs, benchmarkID),
			PolicyID:           policy.ID,
			PolicySeverity:     policy.Severity,
		}

		runnerJob := model.ComplianceRunner{
			BenchmarkID:          rootBenchmarkID,
			QueryID:              *policy.QueryID,
			ConnectionID:         connectionID,
			ResourceCollectionID: resourceCollectionID,
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

	var allRunners []*model.ComplianceRunner
	for _, connection := range assignments.Connections {
		runners, err := s.buildRunners(&connection.ConnectionID, nil, benchmarkID, nil, benchmarkID)
		if err != nil {
			return 0, err
		}
		allRunners = append(allRunners, runners...)
	}

	for _, resourceCollection := range assignments.ResourceCollections {
		runners, err := s.buildRunners(nil, &resourceCollection.ResourceCollectionID, benchmarkID, nil, benchmarkID)
		if err != nil {
			return 0, err
		}
		allRunners = append(allRunners, runners...)
	}

	err = s.db.CreateRunnerJobs(allRunners)
	if err != nil {
		return 0, err
	}

	job := model.ComplianceJob{
		BenchmarkID: benchmarkID,
		Status:      api.ComplianceReportJobCreated,
		IsStack:     false,
	}
	for _, j := range allRunners {
		job.RunnerIDs = append(job.RunnerIDs, int64(j.ID))
	}
	err = s.db.CreateComplianceJob(&job)
	if err != nil {
		return 0, err
	}

	return job.ID, nil
}
