package db

import (
	"github.com/opengovern/opencomply/services/integration/models"
)

// DeleteIntegrationType deletes a credential
func (db Database) DeleteIntegrationType(id string) error {
	tx := db.Orm.
		Where("id = ?", id).
		Unscoped().
		Delete(&models.IntegrationType{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// ListIntegrationTypes list credentials
func (db Database) ListIntegrationTypes() ([]models.IntegrationType, error) {
	var credentials []models.IntegrationType
	tx := db.Orm.
		Model(&models.IntegrationType{}).
		Find(&credentials)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return credentials, nil
}

// GetIntegrationType get a credential
func (db Database) GetIntegrationType(id string) (*models.IntegrationType, error) {
	var integrationType models.IntegrationType
	tx := db.Orm.
		Model(&models.IntegrationType{}).
		Where("id = ?", id).
		First(&integrationType)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &integrationType, nil
}
