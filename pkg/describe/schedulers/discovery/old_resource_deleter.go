package discovery

import (
	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-util/pkg/ticker"
	"go.uber.org/zap"
	"time"
)

const OldResourceDeleterInterval = 1 * time.Minute

func (s *Scheduler) OldResourceDeleter() {
	s.logger.Info("Scheduling OldResourceDeleter on a timer")

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
	s.logger.Info("runDeleter")

	tasks, err := es.GetDeleteTasks(s.esClient)
	if err != nil {
		s.logger.Error("failed to get delete tasks", zap.Error(err))
		return err
	}

	for _, task := range tasks.Hits.Hits {
		for _, resource := range task.Source.DeletingResources {
			err = s.esClient.Delete(string(resource.Key), resource.Index)
			if err != nil {
				s.logger.Error("failed to delete resource", zap.Error(err))
				return err
			}
		}
		err = s.esClient.Delete(task.ID, es.DeleteTasksIndex)
		if err != nil {
			s.logger.Error("failed to delete task", zap.Error(err))
			return err
		}
	}

	return nil
}
