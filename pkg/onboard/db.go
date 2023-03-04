package onboard

import (
	"fmt"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

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
		&SPN{},
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

// ListSources gets list of all source
func (db Database) ListSources() ([]Source, error) {
	var s []Source
	tx := db.orm.Model(Source{}).Preload("Credential").Find(&s)

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
	tx := db.orm.Find(&s, "type = ?", rType)

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
	tx := db.orm.Preload("Credential").First(&s, "id = ?", id.String())

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
	tx := db.orm.
		Model(&Source{}).
		Where("id = ?", id.String()).
		Updates(map[string]interface{}{
			"enabled": enabled,
		})

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("update source: didn't find source to update")
	}

	return nil
}

// CreateSPN creates a new spn
func (db Database) CreateSPN(s *SPN) error {
	tx := db.orm.
		Model(&SPN{}).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(s)

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected == 0 {
		return fmt.Errorf("create spn: didn't create spn due to id conflict")
	}

	return nil
}

// GetSPN gets a spn with matching id
func (db Database) GetSPN(id uuid.UUID) (SPN, error) {
	var s SPN
	tx := db.orm.First(&s, "id = ?", id)

	if tx.Error != nil {
		return SPN{}, tx.Error
	} else if tx.RowsAffected != 1 {
		return SPN{}, gorm.ErrRecordNotFound
	}

	return s, nil
}

// GetSPNByTenantClientID gets a spn with matching clientID and tenantID
func (db Database) GetSPNByTenantClientID(tenantId, clientId string) (SPN, error) {
	var s SPN
	tx := db.orm.First(&s, "tenant_id = ? AND client_id = ?", tenantId, clientId)

	if tx.Error != nil {
		return SPN{}, tx.Error
	} else if tx.RowsAffected != 1 {
		return SPN{}, gorm.ErrRecordNotFound
	}
	return s, nil
}

// DeleteSPN deletes a spn with matching id
func (db Database) DeleteSPN(id uuid.UUID) (SPN, error) {
	var s SPN
	tx := db.orm.Delete(&s, "id = ?", id)

	if tx.Error != nil {
		return SPN{}, tx.Error
	} else if tx.RowsAffected != 1 {
		return SPN{}, gorm.ErrRecordNotFound
	}

	return s, nil
}

// GetAllSPNs lists all spns
func (db Database) GetAllSPNs() ([]SPN, error) {
	var s []SPN
	tx := db.orm.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
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

func (db Database) CountSourcesWithFilters(query interface{}, args ...interface{}) (int64, error) {
	var count int64
	tx := db.orm.Model(&Source{}).Where(query, args).Count(&count)

	if tx.Error != nil {
		return 0, tx.Error
	}

	return count, nil
}

func (db Database) GetCredentialsByConnector(connector source.Type) ([]Credential, error) {
	var creds []Credential
	if connector == source.Nil {
		tx := db.orm.Find(&creds)
		if tx.Error != nil {
			return nil, tx.Error
		}
	} else {
		tx := db.orm.Where("connector = ?", connector).Find(&creds)
		if tx.Error != nil {
			return nil, tx.Error
		}
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
