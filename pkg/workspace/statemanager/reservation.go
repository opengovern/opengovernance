package statemanager

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/sony/sonyflake"
	"math/rand"
)

func (s *Service) UseReservationIfPossible(workspace db.Workspace) error {
	creds, err := s.db.ListCredentialsByWorkspaceID(workspace.ID)
	if err != nil {
		return err
	}

	if len(creds) == 0 {
		return nil
	}

	rs, err := s.db.GetReservedWorkspace(false)
	if err != nil {
		return err
	}

	if rs == nil {
		return nil
	}

	err = s.db.DeleteWorkspace(workspace.ID)
	if err != nil {
		return err
	}

	err = s.db.UpdateCredentialWSID(workspace.ID, rs.ID)
	if err != nil {
		return err
	}

	workspace.ID, rs.ID = rs.ID, workspace.ID
	rs.Name = fmt.Sprintf("rs-deleting-%d", rand.Int())
	rs.Status = api.StateID_Deleting

	err = s.db.UpdateWorkspace(&workspace)
	if err != nil {
		return err
	}

	err = s.db.CreateWorkspace(rs)
	if err != nil {
		return err
	}

	err = s.db.DeleteWorkspaceTransaction(workspace.ID, api.Transaction_CreateHelmRelease)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) handleReservation(ctx context.Context) error {
	rs, err := s.db.GetReservedWorkspace(true)
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

	awsUID, err := sf.NextID()
	if err != nil {
		return err
	}

	workspace := &db.Workspace{
		ID:             fmt.Sprintf("ws-%d", id),
		Name:           "",
		AWSUniqueId:    aws.String(fmt.Sprintf("aws-uid-%d", awsUID)),
		OwnerId:        nil,
		Status:         api.StateID_Reserving,
		Size:           api.SizeXS,
		Tier:           api.Tier_Teams,
		OrganizationID: nil,
	}

	if err := s.db.CreateWorkspace(workspace); err != nil {
		return err
	}
	return nil
}
