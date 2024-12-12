package db

import "github.com/opengovern/opencomply/services/integration/models"

// GetIntegrationTypeSetup Get Integration Type Setup
func (db Database) GetIntegrationTypeSetup(integrationTypeName string) (*models.IntegrationTypeSetup, error) {
	var integrationType models.IntegrationTypeSetup
	tx := db.Orm.
		Model(&models.IntegrationTypeSetup{}).
		Where("integration_type = ?", integrationTypeName).
		First(&integrationType)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &integrationType, nil
}

// ListIntegrationTypeSetup List Integration Type Setup
func (db Database) ListIntegrationTypeSetup() ([]models.IntegrationTypeSetup, error) {
	var integrationType []models.IntegrationTypeSetup
	tx := db.Orm.
		Model(&models.IntegrationTypeSetup{}).
		Find(&integrationType)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return integrationType, nil
}

func (db Database) UpdateIntegrationTypeSetup(integrationTypeSetup *models.IntegrationTypeSetup) error {
	tx := db.Orm.
		Model(&models.IntegrationTypeSetup{}).
		Where("integration_type = ?", integrationTypeSetup.IntegrationType).
		Update("enabled", integrationTypeSetup.Enabled)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) CreateIntegrationTypeSetup(integrationTypeSetup *models.IntegrationTypeSetup) error {
	tx := db.Orm.
		FirstOrCreate(integrationTypeSetup)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}
