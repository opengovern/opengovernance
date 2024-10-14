package db

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/opengovern/og-util/pkg/source"
	"github.com/opengovern/opengovernance/services/integration/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateCredential creates a new credential
func (db Database) CreateCredential(s *model.Credential) error {
	tx := db.Orm.
		Model(&model.Credential{}).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(s)

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected == 0 {
		return fmt.Errorf("create spn: didn't create spn due to id conflict")
	}

	return nil
}

// DeleteCredential deletes a credential
func (db Database) DeleteCredential(id uuid.UUID) error {
	tx := db.Orm.
		Where("id = ?", id.String()).
		Unscoped().
		Delete(&model.Credential{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) GetCredentialsByFilters(connector source.Type, health source.HealthStatus, credentialType []model.CredentialType) ([]model.Credential, error) {
	var creds []model.Credential
	tx := db.Orm.Model(&model.Credential{})
	if connector != source.Nil {
		tx = tx.Where("connector_type = ?", connector)
	}
	if health != source.HealthStatusNil {
		tx = tx.Where("health_status = ?", health)
	}
	if len(credentialType) > 0 {
		tx = tx.Where("credential_type IN ?", credentialType)
	}
	tx = tx.Find(&creds)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return creds, nil
}

func (db Database) GetCredentialByID(id uuid.UUID) (*model.Credential, error) {
	var cred model.Credential
	tx := db.Orm.First(&cred, "id = ?", id)
	if tx.Error != nil {
		return nil, tx.Error
	} else if tx.RowsAffected != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return &cred, nil
}

func (db Database) UpdateCredential(creds *model.Credential) (*model.Credential, error) {
	tx := db.Orm.
		Model(&model.Credential{}).
		Where("id = ?", creds.ID.String()).Updates(creds)

	if tx.Error != nil {
		return nil, tx.Error
	} else if tx.RowsAffected != 1 {
		return nil, fmt.Errorf("update credential: didn't find credential to update")
	}

	return creds, nil

}

func (db Database) DeleteCredentialByID(id uuid.UUID) error {
	tx := db.Orm.
		Where("id = ?", id.String()).
		Unscoped().
		Delete(&model.Credential{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) DeleteCredentials() error {
	tx := db.Orm.
		Where("1 = 1").
		Unscoped().
		Delete(&model.Credential{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
