package describe

import (
	"fmt"
	describeApi "github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
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
		case string(model.JobSequencerJobTypeBenchmark):
			err := s.resolveBenchmarkDependency(job)
			if err != nil {
				s.logger.Error("failed to resolve benchmark dependency", zap.Uint("jobID", job.ID), zap.Error(err))
				if err := s.db.UpdateJobSequencerFailed(job.ID); err != nil {
					return err
				}
				continue
			}
		case string(model.JobSequencerJobTypeDescribe):
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

func (s *Scheduler) runNextJob(job model.JobSequencer) error {
	switch job.NextJob {
	case string(model.JobSequencerJobTypeAnalytics):
		_, err := s.scheduleAnalyticsJob(model.AnalyticsJobTypeNormal)
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

func (s *Scheduler) resolveBenchmarkDependency(job model.JobSequencer) error {
	allDependencyResolved := true
	for _, id := range job.DependencyList {
		complianceJob, err := s.db.GetComplianceJobByID(uint(id))
		if err != nil {
			return err
		}

		if complianceJob == nil {
			return fmt.Errorf("job not found: %v", id)
		}

		if complianceJob.Status == model.ComplianceJobCreated || complianceJob.Status == model.ComplianceJobRunnersInProgress {
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

func (s *Scheduler) resolveDescribeDependency(job model.JobSequencer) error {
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
