package onboard

import (
	"fmt"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Database struct {
	orm *gorm.DB
}

func (db Database) Initialize() error {
	err := db.orm.AutoMigrate(
		&Source{},
		&SPN{},
	)
	if err != nil {
		return err
	}

	return nil
}

// GetSources gets list of all source
func (db Database) GetSources() ([]Source, error) {
	var s []Source
	tx := db.orm.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// CountSources gets list of all source
func (db Database) CountSources() (int64, error) {
	var c int64
	tx := db.orm.Model(&Source{}).Count(&c)

	if tx.Error != nil {
		return 0, tx.Error
	}

	return c, nil
}

// GetSources gets list of sources with matching type
func (db Database) GetSourcesOfType(rType api.SourceType) ([]Source, error) {
	var s []Source
	tx := db.orm.Find(&s, "type = ?", rType)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// CountSources gets list of sources with matching type
func (db Database) CountSourcesOfType(rType api.SourceType) (int64, error) {
	var c int64
	tx := db.orm.Model(&Source{}).Where("type = ?", rType).Count(&c)

	if tx.Error != nil {
		return 0, tx.Error
	}

	return c, nil
}

// GetSource gets a source with matching id
func (db Database) GetSource(id uuid.UUID) (Source, error) {
	var s Source
	tx := db.orm.First(&s, "id = ?", id.String())

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
	} else if tx.RowsAffected != 1 {
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
		Where("id = ?", s.ID.String()).
		Updates(map[string]interface{}{
			"name":       s.Name,
			"config_ref": s.ConfigRef,
			"updated_at": gorm.Expr("NOW()"),
		})

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
		Delete(&Source{})

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("delete source: didn't find source to delete")
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
	} else if tx.RowsAffected != 1 {
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
