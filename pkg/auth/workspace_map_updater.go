package auth

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"go.uber.org/zap"
	"time"
)

func (s *Server) WorkspaceMapUpdater() {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("WorkspaceMapUpdater paniced", zap.Error(fmt.Errorf("%v", r)))
			go s.WorkspaceMapUpdater()
		}
	}()

	for {
		if err := s.updateWorkspaceMap(); err != nil {
			s.logger.Error("failure while updating workspace map", zap.Error(err))
		}
		time.Sleep(5 * time.Minute)
	}
}

func (s *Server) updateWorkspaceMap() error {
	workspaces, err := s.workspaceClient.ListWorkspaces(&httpclient.Context{UserRole: api.InternalRole, UserID: api.GodUserID})
	if err != nil {
		return err
	}

	for _, workspace := range workspaces {
		err = s.db.UpsertWorkspaceMap(workspace.ID, workspace.Name)
		if err != nil {
			s.logger.Error("failed to upsert workspace map", zap.Error(err))
			return err
		}
	}

	workspaceMaps, err := s.db.ListWorkspaceMaps()
	if err != nil {
		s.logger.Error("failed to list workspace maps", zap.Error(err))
		return err
	}
	for _, workspaceMap := range workspaceMaps {
		exists := false
		for _, workspace := range workspaces {
			if workspace.ID == workspaceMap.ID {
				exists = true
			}
		}
		if !exists {
			err = s.db.DeleteWorkspaceMapByID(workspaceMap.ID)
			if err != nil {
				s.logger.Error("failed to delete workspace map", zap.Error(err))
				return err
			}
		}
	}
	return nil
}
