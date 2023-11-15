package statemanager

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/sony/sonyflake"
)

func (s *Service) handleReservation() error {
	sf := sonyflake.NewSonyflake(sonyflake.Settings{})
	id, err := sf.NextID()
	if err != nil {
		return err
	}

	workspace := &db.Workspace{
		ID:             fmt.Sprintf("ws-%d", id),
		Name:           "",
		OwnerId:        nil,
		URI:            "",
		Status:         api.StatusReserved,
		Description:    "",
		Size:           api.SizeXS,
		Tier:           api.Tier_Teams,
		OrganizationID: nil,
	}

	if err := s.db.CreateWorkspace(workspace); err != nil {
		return err
	}
	return nil
}
