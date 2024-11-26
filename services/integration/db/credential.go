package db

import (
	"fmt"
	"github.com/opengovern/opencomply/services/integration/models"
	"gorm.io/gorm/clause"
)

// CreateCredential creates a new credential
func (db Database) CreateCredential(s *models.Credential) error {
	tx := db.Orm.
		Model(&models.Credential{}).
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
func (db Database) DeleteCredential(id string) error {
	tx := db.Orm.
		Where("id = ?", id).
		Unscoped().
		Delete(&models.Credential{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// ListCredentials list credentials
func (db Database) ListCredentials() ([]models.Credential, error) {
	var credentials []models.Credential
	tx := db.Orm.
		Model(&models.Credential{}).
		Find(&credentials)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return credentials, nil
}

// ListCredentialsFiltered list credentials filtered
func (db Database) ListCredentialsFiltered(ids []string, integrationTypes []string) ([]models.Credential, error) {
	var credentials []models.Credential
	tx := db.Orm.
		Model(&models.Credential{})

	if len(ids) > 0 {
		tx = tx.Where("id IN ?", ids)
	}
	if len(integrationTypes) > 0 {
		tx = tx.Where("integration_type IN ?", integrationTypes)
	}

	tx = tx.Find(&credentials)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return credentials, nil
}

// GetCredential get a credential
func (db Database) GetCredential(id string) (*models.Credential, error) {
	var credential models.Credential
	tx := db.Orm.
		Model(&models.Credential{}).
		Where("id = ?", id).
		First(&credential)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &credential, nil
}

func (db Database) UpdateCredential(id string, secret string) error {
	tx := db.Orm.
		Model(&models.Credential{}).
		Where("id = ?", id).Update("secret", secret)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
