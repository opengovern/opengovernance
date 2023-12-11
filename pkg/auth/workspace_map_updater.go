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

	s.mapLock.Lock()
	defer s.mapLock.Unlock()

	for _, workspace := range workspaces {
		s.workspaceIDNameMap[workspace.Name] = workspace.ID
	}
	for name, _ := range s.workspaceIDNameMap {
		exists := false
		for _, workspace := range workspaces {
			if workspace.Name == name {
				exists = true
			}
		}
		if !exists {
			delete(s.workspaceIDNameMap, name)
		}
	}
	return nil
}
