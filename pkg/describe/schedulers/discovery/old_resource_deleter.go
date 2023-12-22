package discovery

import (
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-util/pkg/ticker"
	"go.uber.org/zap"
	"strings"
	"time"
)

const OldResourceDeleterInterval = 1 * time.Minute

func (s *Scheduler) OldResourceDeleter() {
	s.logger.Info("Scheduling compliance summarizer on a timer")

	t := ticker.NewTicker(OldResourceDeleterInterval, time.Second*10)
	defer t.Stop()

	for ; ; <-t.C {
		if err := s.runDeleter(); err != nil {
			s.logger.Error("failed to run deleter", zap.Error(err))
			continue
		}
	}
}

func (s *Scheduler) runDeleter() error {
	deletingJobs, err := s.db.ListDescribeJobsByStatus(api.DescribeResourceJobOldResourceDeletion)
	if err != nil {
		return err
	}

	for _, job := range deletingJobs {
		if job.DeletingCount > 0 {
			tasks, err := es.GetDeleteTasks(s.esClient, job.ID)
			if err != nil {
				return err
			}

			for _, task := range tasks.Hits.Hits {
				for _, resource := range task.Source.DeletingResources {
					err = s.esClient.Delete(string(resource.Key), resource.Index)
					if err != nil {
						if !strings.Contains(err.Error(), "not_found") {
							return err
						}
					}
				}
				err = s.esClient.Delete(task.ID, es.DeleteTasksIndex)

				if err != nil {
					if !strings.Contains(err.Error(), "not_found") {
						return err
					}
				}
			}
		}

		err = s.db.UpdateDescribeConnectionJobStatus(job.ID, api.DescribeResourceJobSucceeded, job.FailureMessage, job.ErrorCode, job.DescribedResourceCount, job.DeletingCount)
		if err != nil {
			return err
		}
	}
	return nil
}
