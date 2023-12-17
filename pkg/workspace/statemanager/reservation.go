package statemanager

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/state"
	"github.com/sony/sonyflake"
)

func (s *Service) handleReservation() error {
	rs, err := s.db.GetReservedWorkspace()
	if err != nil {
		return err
	}

	if rs != nil {
		//if workspace.Status != "RESERVING" && workspace.Status != "RESERVED" {
		//	rs, err := t.db.GetReservedWorkspace()
		//	if err != nil {
		//		return err
		//	}
		//
		//	if rs != nil {
		//		err = t.db.DeleteWorkspace(workspace.ID)
		//		if err != nil {
		//			return err
		//		}
		//
		//		err = t.db.UpdateCredentialWSID(workspace.ID, rs.ID)
		//		if err != nil {
		//			return err
		//		}
		//
		//		workspace.ID = rs.ID
		//		if err := t.db.UpdateWorkspace(&workspace); err != nil {
		//			return err
		//		}
		//
		//		return ErrTransactionNeedsTime
		//	}
		//}

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
		Status:         string(state.StateID_Reserving),
		Size:           api.SizeXS,
		Tier:           api.Tier_Teams,
		OrganizationID: nil,
	}

	if err := s.db.CreateWorkspace(workspace); err != nil {
		return err
	}
	return nil
}
