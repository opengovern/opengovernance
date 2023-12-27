package describe

import (
	"context"
	"errors"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
)

var ErrMaxResourceCountExceeded = errors.New("maximum resource count exceeded")

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
		return ErrMaxResourceCountExceeded
	}

	return nil
}
