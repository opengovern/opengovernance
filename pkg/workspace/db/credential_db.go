package db

import (
	"errors"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/gorm"
)

func (s *Database) CreateMasterCredential(cred *MasterCredential) error {
	err := s.Orm.Model(&MasterCredential{}).
		Create(cred).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Database) GetMasterCredentialByWorkspaceID(workspaceID string) (*MasterCredential, error) {
	var res MasterCredential
	err := s.Orm.Model(&MasterCredential{}).Where("workspace_id = ?", workspaceID).Find(&res).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (s *Database) ListCredentialsByWorkspaceID(id string) ([]Credential, error) {
	var creds []Credential
	err := s.Orm.Model(&Credential{}).
		Where("workspace_id = ?", id).
		Find(&creds).Error
	if err != nil {
		return nil, err
	}
	return creds, nil
}

func (s *Database) CreateCredential(cred *Credential) error {
	err := s.Orm.Model(&Credential{}).
		Create(cred).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Database) CountConnectionsByConnector(workspaceID string, connector source.Type) (int64, error) {
	var count int64
	tx := s.Orm.Raw("select coalesce(sum(connection_count),0) from credentials where workspace_id = ? and connector_type = ?", workspaceID, connector).Find(&count)
	err := tx.Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Database) SetIsCreated(id uint) error {
	tx := s.Orm.
		Model(&Credential{}).
		Where("id = ?", id).
		Update("is_created", true)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (s *Database) DeleteCredential(id uint) error {
	tx := s.Orm.
		Where("id = ?", id).
		Unscoped().
		Delete(&Credential{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
