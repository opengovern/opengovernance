package describe

import (
	"context"
	"errors"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"

	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
)

func (s *Scheduler) CheckWorkspaceResourceLimit() error {
	limit, err := s.workspaceClient.GetLimitsByID(&httpclient.Context{
		UserRole: api2.ViewerRole,
	}, CurrentWorkspaceID)
	if err != nil {
		return err
	}

	currentResourceCount, err := s.es.Count(context.Background(), InventorySummaryIndex)
	if err != nil {
		return err
	}

	if currentResourceCount >= limit.MaxResources {
		return errors.New("maximum resource count exceeded")
	}

	if err = s.rdb.Set(context.Background(), RedisKeyWorkspaceResourceRemaining,
		limit.MaxResources-currentResourceCount, 12*time.Hour).Err(); err != nil {
		return err
	}

	return nil
}
