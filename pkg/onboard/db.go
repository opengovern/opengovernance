package onboard

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Database struct {
	orm *gorm.DB
}

// GetOrganization gets an organization with maptching id
func (db Database) GetOrganization(id uuid.UUID) (*Organization, error) {
	var o Organization
	tx := db.orm.
		First(&o, "id = ? AND deleted_at IS NULL", id.String())

	if tx.Error != nil {
		return nil, tx.Error
	} else if tx.RowsAffected != 1 {
		return nil, fmt.Errorf("get organization: specified id was not found")
	}

	return &o, nil
}

// CreateOrganization creates a new organization and returns it
func (db Database) CreateOrganization(o *Organization) (*Organization, error) {
	tx := db.orm.
		Model(&Organization{}).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(o)

	if tx.Error != nil {
		return nil, tx.Error
	} else if tx.RowsAffected != 1 {
		return nil, fmt.Errorf("create organization: didn't create organization due to id conflict")
	}

	return o, nil
}

// UpdateOrganization updates an existing organization and returns it
func (db Database) UpdateOrganization(o *Organization) (*Organization, error) {
	tx := db.orm.
		Model(&Organization{}).
		Where("id = ?", o.ID.String()).
		Updates(map[string]interface{}{
			"name":        o.Name,
			"description": o.Description,
			"admin_email": o.AdminEmail,
			"keibi_url":   o.KeibiUrl,
			"vault_ref":   o.VaultRef,
			"updated_at":  gorm.Expr("NOW() AT TIME ZONE 'UTC'"),
		})

	if tx.Error != nil {
		return nil, tx.Error
	} else if tx.RowsAffected != 1 {
		return nil, fmt.Errorf("update organization: didn't find organization to update")
	}

	return o, nil
}

// DeleteOrganization deletes an existing organization
func (db Database) DeleteOrganization(id uuid.UUID) error {
	tx := db.orm.
		Where("id = ?", id.String()).
		Delete(&Organization{})

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("delete organization: didn't find organization to delete")
	}

	return nil
}

// GetSource gets a source with matching id
func (db Database) GetSource(id uuid.UUID) (*Source, error) {
	var s Source
	tx := db.orm.Joins("AWSMetadata").First(&s, "sources.id = ?", id.String())

	if tx.Error != nil {
		return nil, tx.Error
	} else if tx.RowsAffected != 1 {
		return nil, fmt.Errorf("get source: specified id was not found")
	}

	return &s, nil
}

// CreateSource creates a new source and returns it
func (db Database) CreateSource(s *Source) (*Source, error) {
	tx := db.orm.
		Model(&Source{}).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(s)

	if tx.Error != nil {
		return nil, tx.Error
	} else if tx.RowsAffected != 1 {
		return nil, fmt.Errorf("create source: didn't create source due to id conflict")
	}

	return s, nil
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
