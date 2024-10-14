package db

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/opengovern/og-util/pkg/source"
	"github.com/opengovern/opengovernance/services/integration/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

// ListSources gets list of all source
func (db Database) ListSources() ([]model.Connection, error) {
	var s []model.Connection
	tx := db.Orm.Model(model.Connection{}).Preload(clause.Associations).Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// ListSourcesWithFilters gets list of all source with specified filters
func (db Database) ListSourcesWithFilters(
	connectorTypes []source.Type,
	connectionIDs []string,
	lifecycleState []model.ConnectionLifecycleState,
	healthState []source.HealthStatus) ([]model.Connection, error) {

	var s []model.Connection
	tx := db.Orm.Model(model.Connection{}).Preload(clause.Associations)
	if len(connectorTypes) > 0 {
		tx = tx.Where("type IN ?", connectorTypes)
	}
	if len(connectionIDs) > 0 {
		tx = tx.Where("id IN ?", connectionIDs)
	}
	if len(lifecycleState) > 0 {
		tx = tx.Where("lifecycle_state IN ?", lifecycleState)
	}
	if len(healthState) > 0 {
		tx = tx.Where("health_state IN ?", healthState)
	}
	tx.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

// GetSources gets sources by id
func (db Database) GetSources(ids []string) ([]model.Connection, error) {
	var s []model.Connection
	tx := db.Orm.Preload(clause.Associations).Find(&s, "id in ?", ids)

	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

// CountSources gets count of all source
func (db Database) CountSources() (int64, error) {
	var c int64
	tx := db.Orm.Model(&model.Connection{}).Count(&c)

	if tx.Error != nil {
		return 0, tx.Error
	}

	return c, nil
}

// GetSourcesOfType gets list of sources with matching type
func (db Database) GetSourcesOfType(rType source.Type) ([]model.Connection, error) {
	var s []model.Connection
	tx := db.Orm.Preload(clause.Associations).Find(&s, "type = ?", rType)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// GetSourcesOfTypes gets list of sources with matching types
func (db Database) GetSourcesOfTypes(rTypes []source.Type) ([]model.Connection, error) {
	var s []model.Connection
	tx := db.Orm.Preload(clause.Associations).Find(&s, "type IN ?", rTypes)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// CountSourcesOfType gets count of sources with matching type
func (db Database) CountSourcesOfType(rType source.Type) (int64, error) {
	var c int64
	tx := db.Orm.Model(&model.Connection{}).Where("type = ?", rType.String()).Count(&c)

	if tx.Error != nil {
		return 0, tx.Error
	}

	return c, nil
}

// GetSource gets a source with matching id
func (db Database) GetSource(id uuid.UUID) (model.Connection, error) {
	var s model.Connection
	tx := db.Orm.Preload(clause.Associations).First(&s, "id = ?", id.String())

	if tx.Error != nil {
		return model.Connection{}, tx.Error
	} else if tx.RowsAffected != 1 {
		return model.Connection{}, gorm.ErrRecordNotFound
	}

	return s, nil
}

// GetSourceBySourceID gets a source with matching source id
func (db Database) GetSourceBySourceID(id string) (model.Connection, error) {
	var s model.Connection
	tx := db.Orm.Model(&model.Connection{}).Where("source_id = ?", id).First(&s)

	if tx.Error != nil {
		return model.Connection{}, tx.Error
	}
	return s, nil
}

func (db Database) ListSourcesByFilters(connector, providerNameRegex, providerIdRegex *string, healthState *bool) ([]model.Connection, error) {
	var s []model.Connection
	tx := db.Orm.Model(&model.Connection{})

	if connector != nil {
		tx = tx.Where("type = ?", connector)
	}
	if providerNameRegex != nil {
		tx = tx.Where("name ~* ?", providerNameRegex)
	}
	if providerIdRegex != nil {
		tx = tx.Where("source_id ~* ?", *providerIdRegex)
	}
	if healthState != nil {
		if *healthState {
			tx = tx.Where("health_state = ?", source.HealthStatusHealthy)
		} else {
			tx = tx.Where("health_state = ?", source.HealthStatusUnhealthy)
		}
	}

	tx = tx.Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) ListIntegrationsFiltered(integrationTracker, connector []string, nameRegex, idRegex string, healthState source.HealthStatus) ([]model.Connection, error) {
	var s []model.Connection
	tx := db.Orm.Model(&model.Connection{})

	if len(connector) > 0 {
		tx = tx.Where("type IN ?", connector)
	}
	if nameRegex != "" {
		tx = tx.Where("name ~* ?", nameRegex)
	}
	if idRegex != "" {
		tx = tx.Where("source_id ~* ?", idRegex)
	}
	if healthState != "" {
		tx = tx.Where("health_state = ?", healthState)
	}
	if len(integrationTracker) > 0 {
		tx = tx.Where("id IN ?", integrationTracker)
	}

	tx = tx.Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) GetSourceByFilters(connector, providerName, providerId *string) (model.Connection, error) {
	var s model.Connection
	tx := db.Orm.Model(&model.Connection{})

	if connector != nil {
		tx = tx.Where("type = ?", connector)
	}
	if providerName != nil {
		tx = tx.Where("name = ?", providerName)
	}
	if providerId != nil {
		tx = tx.Where("source_id = ?", *providerId)
	}

	tx = tx.First(&s)
	if tx.Error != nil {
		return model.Connection{}, tx.Error
	}
	return s, nil
}

// GetSourcesByCredentialID list sources with matching credential id
func (db Database) GetSourcesByCredentialID(id string) ([]model.Connection, error) {
	var s []model.Connection
	tx := db.Orm.Find(&s, "credential_id = ?", id)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

// CreateSource creates a new source and returns it
func (db Database) CreateSource(s *model.Connection) error {
	tx := db.Orm.
		Model(&model.Connection{}).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(s)

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("create source: didn't create source due to id conflict")
	}

	return nil
}

// UpdateSource updates an existing source and returns it
func (db Database) UpdateSource(s *model.Connection) (*model.Connection, error) {
	tx := db.Orm.
		Model(&model.Connection{}).
		Where("id = ?", s.ID.String()).Updates(s)

	if tx.Error != nil {
		return nil, tx.Error
	} else if tx.RowsAffected != 1 {
		return nil, fmt.Errorf("update source: didn't find source to update")
	}

	return s, nil
}

func (db Database) DeleteSource(id uuid.UUID) error {
	tx := db.Orm.
		Where("id = ?", id.String()).
		Unscoped().
		Delete(&model.Connection{})

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("delete source: didn't find source to delete")
	}

	return nil
}

// UpdateSourceLifecycleState update source lifecycle state
func (db Database) UpdateSourceLifecycleState(id uuid.UUID, state model.ConnectionLifecycleState) error {
	tx := db.Orm.
		Model(&model.Connection{}).
		Where("id = ?", id.String()).
		Updates(map[string]interface{}{
			"lifecycle_state": state,
		})

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("update source: didn't find source to update")
	}

	return nil
}

func (db Database) CountSourcesWithFilters(query string, args ...interface{}) (int64, error) {
	var count int64
	tx := db.Orm.Model(&model.Connection{})
	if len(args) > 0 {
		tx = tx.Where(query, args)
	} else if len(strings.TrimSpace(query)) > 0 {
		tx = tx.Where(query)
	}
	tx = tx.Count(&count)

	if tx.Error != nil {
		return 0, tx.Error
	}

	return count, nil
}

func (db Database) CountConnectionsByCredential(credentialId string, state []model.ConnectionLifecycleState, healthStates []source.HealthStatus) (int, error) {
	var count int64
	tx := db.Orm.Model(&model.Connection{}).Where("credential_id = ?", credentialId)
	if len(state) > 0 {
		tx = tx.Where("lifecycle_state IN ?", state)
	}
	if len(healthStates) > 0 {
		tx = tx.Where("health_state IN ?", healthStates)
	}
	tx = tx.Count(&count)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return int(count), nil
}

func (db Database) DeleteSources() error {
	tx := db.Orm.Where("1 = 1").Unscoped().Delete(&model.Connection{})

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
