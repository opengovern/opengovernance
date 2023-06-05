package onboard

import (
	"fmt"
	"strings"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Database struct {
	orm *gorm.DB
}

func (db Database) Initialize() error {
	err := db.orm.AutoMigrate(
		&Connector{},
		&Credential{},
		&Source{},
	)
	if err != nil {
		return err
	}

	return nil
}

// ListConnectors gets list of all connectors
func (db Database) ListConnectors() ([]Connector, error) {
	var s []Connector
	tx := db.orm.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// GetConnector gets connector by name
func (db Database) GetConnector(name source.Type) (Connector, error) {
	var c Connector
	tx := db.orm.First(&c, "name = ?", name)

	if tx.Error != nil {
		return Connector{}, tx.Error
	}

	return c, nil
}

// ListSources gets list of all source
func (db Database) ListSources() ([]Source, error) {
	var s []Source
	tx := db.orm.Model(Source{}).Preload(clause.Associations).Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// ListSourcesWithFilters gets list of all source with specified filters
func (db Database) ListSourcesWithFilters(
	connectorTypes []source.Type,
	connectionIDs []string,
	healthStates []source.HealthStatus,
	lifecycleState []ConnectionLifecycleState) ([]Source, error) {

	var s []Source
	tx := db.orm.Model(Source{}).Preload(clause.Associations)
	if len(connectorTypes) > 0 {
		tx = tx.Where("type IN ?", connectorTypes)
	}
	if len(connectionIDs) > 0 {
		tx = tx.Where("id IN ?", connectionIDs)
	}
	if len(healthStates) > 0 {
		tx = tx.Where("health_state IN ?", connectionIDs)
	}
	if len(lifecycleState) > 0 {
		tx = tx.Where("lifecycle_state IN ?", lifecycleState)
	}
	tx.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

// GetSources gets sources by id
func (db Database) GetSources(ids []uuid.UUID) ([]Source, error) {
	var s []Source
	tx := db.orm.Find(&s, "id in ?", ids)

	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

// CountSources gets count of all source
func (db Database) CountSources() (int64, error) {
	var c int64
	tx := db.orm.Model(&Source{}).Count(&c)

	if tx.Error != nil {
		return 0, tx.Error
	}

	return c, nil
}

// GetSourcesOfType gets list of sources with matching type
func (db Database) GetSourcesOfType(rType source.Type) ([]Source, error) {
	var s []Source
	tx := db.orm.Preload(clause.Associations).Find(&s, "type = ?", rType)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// GetSourcesOfTypes gets list of sources with matching types
func (db Database) GetSourcesOfTypes(rTypes []source.Type) ([]Source, error) {
	var s []Source
	tx := db.orm.Preload(clause.Associations).Find(&s, "type IN ?", rTypes)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// CountSourcesOfType gets count of sources with matching type
func (db Database) CountSourcesOfType(rType source.Type) (int64, error) {
	var c int64
	tx := db.orm.Model(&Source{}).Where("type = ?", rType.String()).Count(&c)

	if tx.Error != nil {
		return 0, tx.Error
	}

	return c, nil
}

// GetSource gets a source with matching id
func (db Database) GetSource(id uuid.UUID) (Source, error) {
	var s Source
	tx := db.orm.Preload(clause.Associations).First(&s, "id = ?", id.String())

	if tx.Error != nil {
		return Source{}, tx.Error
	} else if tx.RowsAffected != 1 {
		return Source{}, gorm.ErrRecordNotFound
	}

	return s, nil
}

// GetSourceBySourceID gets a source with matching source id
func (db Database) GetSourceBySourceID(id string) (Source, error) {
	var s Source
	tx := db.orm.First(&s, "source_id = ?", id)

	if tx.Error != nil {
		return Source{}, tx.Error
	} else if s.SourceId != id {
		return Source{}, gorm.ErrRecordNotFound
	}
	return s, nil
}

// GetSourcesByCredentialID list sources with matching credential id
func (db Database) GetSourcesByCredentialID(id string) ([]Source, error) {
	var s []Source
	tx := db.orm.Find(&s, "credential_id = ?", id)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

// CreateSource creates a new source and returns it
func (db Database) CreateSource(s *Source) error {
	tx := db.orm.
		Model(&Source{}).
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
func (db Database) UpdateSource(s *Source) (*Source, error) {
	tx := db.orm.
		Model(&Source{}).
		Where("id = ?", s.ID.String()).Updates(s)

	if tx.Error != nil {
		return nil, tx.Error
	} else if tx.RowsAffected != 1 {
		return nil, fmt.Errorf("update source: didn't find source to update")
	}

	return s, nil
}

// DeleteSource deletes an existing source
func (s *Source) BeforeDelete(tx *gorm.DB) error {
	t := tx.Model(&Source{}).
		Where("id = ?", s.ID.String()).
		Update("lifecycle_state", ConnectionLifecycleStateDeleted)
	return t.Error
}

func (db Database) DeleteSource(id uuid.UUID) error {
	tx := db.orm.
		Where("id = ?", id.String()).
		Unscoped().
		Delete(&Source{})

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("delete source: didn't find source to delete")
	}

	return nil
}

// UpdateSourceEnabled update source enabled
func (db Database) UpdateSourceEnabled(id uuid.UUID, enabled bool) error {
	nextState := ConnectionLifecycleStateDisabled
	if enabled {
		nextState = ConnectionLifecycleStateEnabled
	}

	tx := db.orm.
		Model(&Source{}).
		Where("id = ?", id.String()).
		Updates(map[string]interface{}{
			"enabled":         enabled,
			"lifecycle_state": nextState,
		})

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("update source: didn't find source to update")
	}

	return nil
}

// CreateCredential creates a new credential
func (db Database) CreateCredential(s *Credential) error {
	tx := db.orm.
		Model(&Credential{}).
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
	tx := db.orm.
		Where("id = ?", id.String()).
		Unscoped().
		Delete(&Credential{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) CountSourcesWithFilters(query string, args ...interface{}) (int64, error) {
	var count int64
	tx := db.orm.Model(&Source{})
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

func (db Database) GetCredentialsByFilters(connector source.Type, health source.HealthStatus) ([]Credential, error) {
	var creds []Credential
	tx := db.orm.Model(&Credential{})
	if connector != source.Nil {
		tx = tx.Where("connector_type = ?", connector)
	}
	if health != source.HealthStatusNil {
		tx = tx.Where("health_status = ?", health)
	}
	tx = tx.Find(&creds)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return creds, nil
}

func (db Database) GetCredentialByID(id uuid.UUID) (*Credential, error) {
	var cred Credential
	tx := db.orm.First(&cred, "id = ?", id)
	if tx.Error != nil {
		return nil, tx.Error
	} else if tx.RowsAffected != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return &cred, nil
}

func (db Database) UpdateCredential(creds *Credential) (*Credential, error) {
	tx := db.orm.
		Model(&Credential{}).
		Where("id = ?", creds.ID.String()).Updates(creds)

	if tx.Error != nil {
		return nil, tx.Error
	} else if tx.RowsAffected != 1 {
		return nil, fmt.Errorf("update credential: didn't find credential to update")
	}

	return creds, nil

}

func (db Database) DeleteCredentialByID(id uuid.UUID) error {
	tx := db.orm.
		Where("id = ?", id.String()).
		Unscoped().
		Delete(&Credential{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
