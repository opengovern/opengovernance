package describe

import (
	"strings"
	"time"

	"go.uber.org/zap"
)

func (s Scheduler) cleanupDescribeJob() {
	latestSuccessfulJobsMap, err := s.db.GetLatestSuccessfulDescribeJobIDsPerResourcePerAccount()
	if err != nil {
		s.logger.Error("Failed to get latest successful DescribeResourceJobs per resource per account",
			zap.Error(err),
		)
		DescribeCleanupJobsCount.WithLabelValues("failure").Inc()
		return
	}

	for resourceType, jobIDs := range latestSuccessfulJobsMap {
		s.enqueueExclusiveCleanupJob(resourceType, jobIDs)
	}

	DescribeCleanupJobsCount.WithLabelValues("successful").Inc()
}

func (s Scheduler) enqueueExclusiveCleanupJob(resourceType string, jobIDs []uint) {
	if isPublishingBlocked(s.logger, s.describeCleanupJobQueue) {
		s.logger.Warn("The jobs in queue is over the threshold")
		return
	}

	if err := s.describeCleanupJobQueue.Publish(DescribeCleanupJob{
		JobType:      DescribeCleanupJobTypeExclusiveDelete,
		JobIDs:       jobIDs,
		ResourceType: strings.ToLower(resourceType),
	}); err != nil {
		s.logger.Error("Failed to publish describe clean up job to queue",
			zap.Error(err),
		)
		DescribeCleanupJobsCount.WithLabelValues("failure").Inc()
		return
	}

	DescribeCleanupJobsCount.WithLabelValues("successful").Inc()
}

func (s Scheduler) handleConnectionDescribeJobsCleanup(jobs []DescribeSourceJob) {
	for _, sj := range jobs {
		// I purposefully didn't embbed this query in the previous query to keep returned results count low.
		drj, err := s.db.ListDescribeResourceJobs(sj.ID)
		if err != nil {
			s.logger.Error("Failed to retrieve DescribeResourceJobs for DescribeSouceJob",
				zap.Uint("jobId", sj.ID),
				zap.Error(err),
			)
			DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
			continue
		}

		success := true
		for _, rj := range drj {
			if isPublishingBlocked(s.logger, s.describeCleanupJobQueue) {
				s.logger.Warn("The jobs in queue is over the threshold")
				return
			}

			if err := s.describeCleanupJobQueue.Publish(DescribeCleanupJob{
				JobType:      DescribeCleanupJobTypeInclusiveDelete,
				JobIDs:       []uint{rj.ID},
				ResourceType: rj.ResourceType,
			}); err != nil {
				s.logger.Error("Failed to publish describe clean up job to queue for DescribeResourceJob",
					zap.Uint("jobId", rj.ID),
					zap.Error(err),
				)
				success = false
				DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
				continue
			}

			err = s.db.DeleteDescribeResourceJob(rj.ID)
			if err != nil {
				s.logger.Error("Failed to delete DescribeResourceJob",
					zap.Uint("jobId", rj.ID),
					zap.Error(err),
				)
				success = false
				DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
				continue
			}
		}

		if success {
			err := s.db.DeleteDescribeSourceJob(sj.ID)
			if err != nil {
				s.logger.Error("Failed to delete DescribeSourceJob",
					zap.Uint("jobId", sj.ID),
					zap.Error(err),
				)
				DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
			} else {
				DescribeCleanupSourceJobsCount.WithLabelValues("successful").Inc()
			}
		} else {
			DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
		}

		s.logger.Info("Successfully deleted DescribeSourceJob and its DescribeResourceJobs",
			zap.Uint("jobId", sj.ID),
		)
	}
}

func (s *Scheduler) RunDescribeCleanupJobScheduler() {
	s.logger.Info("Running describe cleanup job scheduler")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for range t.C {
		s.cleanupDescribeJob()
	}
}
