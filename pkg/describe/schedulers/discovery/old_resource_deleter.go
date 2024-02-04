package discovery

import (
	"context"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	es2 "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/ticker"

	"go.uber.org/zap"
	"strings"
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
		switch task.Source.TaskType {
		case es.DeleteTaskTypeResource:
			for _, resource := range task.Source.DeletingResources {
				err = s.esClient.Delete(string(resource.Key), resource.Index)
				if err != nil {
					if strings.Contains(err.Error(), "[404 Not Found]") {
						s.logger.Warn("resource not found", zap.String("resource", string(resource.Key)), zap.String("index", resource.Index), zap.Error(err))
						continue
					}
					s.logger.Error("failed to delete resource", zap.Error(err))
					return err
				}
			}
		case es.DeleteTaskTypeQuery:
			_, err = es2.DeleteByQuery(context.Background(), s.esClient.ES(), []string{task.Source.QueryIndex}, task.Source.Query)
			if err != nil {
				s.logger.Error("failed to delete by query", zap.Error(err))
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
