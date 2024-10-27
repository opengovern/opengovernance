package db

import (
	"fmt"
	"github.com/google/uuid"
	integration_type "github.com/opengovern/opengovernance/services/integration-v2/integration-type"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
	"gorm.io/gorm/clause"
)

// CreateIntegration creates a new integration
func (db Database) CreateIntegration(s *models.Integration) error {
	tx := db.Orm.
		Model(&models.Integration{}).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(s)

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected == 0 {
		return fmt.Errorf("create spn: didn't create spn due to id conflict")
	}

	return nil
}

// DeleteIntegration deletes a integration
func (db Database) DeleteIntegration(integrationTracker uuid.UUID) error {
	tx := db.Orm.
		Where("integration_tracker = ?", integrationTracker.String()).
		Unscoped().
		Delete(&models.Integration{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// ListIntegration list Integration
func (db Database) ListIntegration(types []integration_type.IntegrationType) ([]models.Integration, error) {
	var integrations []models.Integration
	tx := db.Orm.
		Model(&models.Integration{})

	if len(types) > 0 {
		tx = tx.Where("type IN ?", types)
	}

	tx = tx.Find(&integrations)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return integrations, nil
}

// GetIntegration get a Integration
func (db Database) GetIntegration(tracker uuid.UUID) (*models.Integration, error) {
	var integration models.Integration
	tx := db.Orm.
		Model(&models.Integration{}).
		Where("integration_tracker = ?", tracker.String()).
		First(&integration)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &integration, nil
}

// UpdateIntegration deletes a integration
func (db Database) UpdateIntegration(integration *models.Integration) error {
	tx := db.Orm.
		Where("integration_tracker = ?", integration.IntegrationTracker.String()).
		Updates(integration)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
