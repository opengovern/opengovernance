package statemanager

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"strconv"
	"time"
)

func (s *Service) handleAutoSuspend(workspace *db.Workspace) error {
	if workspace.Tier != api.Tier_Free {
		return nil
	}
	switch api.WorkspaceStatus(workspace.Status) {
	case api.StatusDeleting, api.StatusDeleted:
		return nil
	}

	fmt.Printf("checking for auto-suspend %s\n", workspace.Name)

	res, err := s.rdb.Get(context.Background(), "last_access_"+workspace.Name).Result()
	if err != nil {
		if err != redis.Nil {
			return fmt.Errorf("get last access: %v", err)
		}
	}
	lastAccess, _ := strconv.ParseInt(res, 10, 64)
	fmt.Printf("last access: %d [%s]\n", lastAccess, res)

	if time.Now().UnixMilli()-lastAccess > s.cfg.AutoSuspendDuration.Milliseconds() {
		if workspace.Status == api.StatusProvisioned {
			fmt.Printf("suspending workspace %s\n", workspace.Name)
			if err := s.db.UpdateWorkspaceStatus(workspace.ID, api.StatusSuspending); err != nil {
				return fmt.Errorf("update workspace status: %w", err)
			}
		}
	} /* else {
		if workspace.Status == string(StatusSuspended) {
			fmt.Printf("resuming workspace %s\n", workspace.Name)
			if err := s.db.UpdateWorkspaceStatus(workspace.ID, StatusProvisioning); err != nil {
				return fmt.Errorf("update workspace status: %w", err)
			}
		}
	}*/
	return nil
}
