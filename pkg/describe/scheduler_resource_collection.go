package describe

import (
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"go.uber.org/zap"
	"time"
)

func (s *Scheduler) RunResourceCollectionsAnalyticsJobScheduler() {
	s.logger.Info("Scheduling analytics jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		resourceCollections, err := s.inventoryClient.ListResourceCollections(&httpclient.Context{UserRole: authApi.InternalRole})
		if err != nil {
			s.logger.Error("Failed to list resource collections", zap.Error(err))
			continue
		}
		for _, resourceCollection := range resourceCollections {
			resourceCollection := resourceCollection
			lastJob, err := s.db.FetchLastAnalyticsJobForCollectionId(&resourceCollection.ID)
			if err != nil {
				s.logger.Error("Failed to find the last job to check for AnalyticsJob on resourceCollection", zap.Error(err), zap.String("resourceCollectionId", resourceCollection.ID))
				AnalyticsJobsCount.WithLabelValues("failure").Inc()
				continue
			}
			if lastJob == nil || lastJob.CreatedAt.Add(time.Duration(s.analyticsIntervalHours)*time.Hour).Before(time.Now()) {
				err := s.scheduleAnalyticsJob(&resourceCollection.ID)
				if err != nil {
					s.logger.Error("failure on scheduleAnalyticsJob", zap.Error(err))
				}
			}
		}
	}
}
