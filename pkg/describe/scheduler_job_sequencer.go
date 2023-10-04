package describe

import (
	"fmt"
	complianceApi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	describeApi "github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"go.uber.org/zap"
	"time"
)

func (s *Scheduler) RunJobSequencer() {
	s.logger.Info("Scheduling job sequencer")

	t := time.NewTicker(JobSequencerInterval)
	defer t.Stop()

	for ; ; <-t.C {
		err := s.checkJobSequences()
		if err != nil {
			s.logger.Error("failed to run checkJobSequences", zap.Error(err))
			continue
		}
	}
}

func (s *Scheduler) checkJobSequences() error {
	jobs, err := s.db.ListWaitingJobSequencers()
	if err != nil {
		return err
	}

	for _, job := range jobs {
		switch job.DependencySource {
		case string(JobSequencerJobTypeBenchmark):
			err := s.resolveBenchmarkDependency(job)
			if err != nil {
				s.logger.Error("failed to resolve benchmark dependency", zap.Uint("jobID", job.ID), zap.Error(err))
				if err := s.db.UpdateJobSequencerFailed(job.ID); err != nil {
					return err
				}
				continue
			}
		case string(JobSequencerJobTypeDescribe):
			err := s.resolveDescribeDependency(job)
			if err != nil {
				s.logger.Error("failed to resolve describe dependency", zap.Uint("jobID", job.ID), zap.Error(err))
				if err := s.db.UpdateJobSequencerFailed(job.ID); err != nil {
					return err
				}
				continue
			}
		default:
			s.logger.Error("job dependency %s not supported", zap.Uint("jobID", job.ID), zap.String("dependencySource", job.DependencySource))
		}
	}
	return nil
}

func (s *Scheduler) runNextJob(job JobSequencer) error {
	switch job.NextJob {
	case string(JobSequencerJobTypeBenchmarkSummarizer):
		err := s.scheduleComplianceSummarizerJob()
		if err != nil {
			return err
		}

		err = s.db.UpdateJobSequencerFinished(job.ID)
		if err != nil {
			return err
		}
	case string(JobSequencerJobTypeAnalytics):
		err := s.scheduleAnalyticsJob()
		if err != nil {
			return err
		}

		err = s.db.UpdateJobSequencerFinished(job.ID)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("job type %s not supported", job.NextJob)
	}
	return nil
}

func (s *Scheduler) resolveBenchmarkDependency(job JobSequencer) error {
	allDependencyResolved := true
	for _, id := range job.DependencyList {
		complianceJob, err := s.db.GetComplianceReportJobByID(uint(id))
		if err != nil {
			return err
		}

		if complianceJob == nil {
			return fmt.Errorf("job not found: %v", id)
		}

		if complianceJob.Status == complianceApi.ComplianceReportJobCreated || complianceJob.Status == complianceApi.ComplianceReportJobInProgress {
			allDependencyResolved = false
			break
		}
	}

	if allDependencyResolved {
		err := s.runNextJob(job)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) resolveDescribeDependency(job JobSequencer) error {
	allDependencyResolved := true
	for _, id := range job.DependencyList {
		describeConnectionJob, err := s.db.GetDescribeConnectionJobByID(uint(id))
		if err != nil {
			return err
		}

		if describeConnectionJob == nil {
			return fmt.Errorf("job not found: %v", id)
		}

		if describeConnectionJob.Status != describeApi.DescribeResourceJobSucceeded &&
			describeConnectionJob.Status != describeApi.DescribeResourceJobFailed &&
			describeConnectionJob.Status != describeApi.DescribeResourceJobTimeout {
			allDependencyResolved = false
			break
		}
	}

	if allDependencyResolved {
		err := s.runNextJob(job)
		if err != nil {
			return err
		}
	}
	return nil
}
