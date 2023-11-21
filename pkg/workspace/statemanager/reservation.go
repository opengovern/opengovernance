package statemanager

import (
	"context"
	"fmt"
	aws2 "github.com/kaytu-io/kaytu-aws-describer/aws"
	workspace2 "github.com/kaytu-io/kaytu-engine/pkg/workspace"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/sony/sonyflake"
)

func (s *Service) handleReservation() error {
	rs, err := s.db.GetReservedWorkspace()
	if err != nil {
		return err
	}

	if rs != nil {
		return nil
	}

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

	awsConfig, err := aws2.GetConfig(context.Background(), s.cfg.AWSMasterAccessKey, s.cfg.AWSMasterSecretKey, "", "", nil)
	if err != nil {
		return err
	}

	userARN, err := workspace2.CreateOrGetUser(awsConfig, fmt.Sprintf("kaytu-user-%s", workspace.ID))
	if err != nil {
		return err
	}
	workspace.AWSUserARN = &userARN

	if err := s.db.CreateWorkspace(workspace); err != nil {
		return err
	}
	return nil
}
