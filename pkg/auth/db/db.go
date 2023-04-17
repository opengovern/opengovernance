package db

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gorm.io/gorm"
)

type Database struct {
	Orm *gorm.DB
}

func (db Database) Initialize() error {
	err := db.Orm.AutoMigrate(
		&ApiKey{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (db Database) ListApiKeys(workspaceID string) ([]ApiKey, error) {
	var s []ApiKey
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("revoked", "false").
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) GetAPIKeysByRole(role api.Role, workspaceID string) ([]ApiKey, error) {
	var s []ApiKey
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("role", role).
		Where("revoked", "false").
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) CountApiKeys(workspaceID string) (int64, error) {
	var s int64
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("revoked", "false").
		Count(&s)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return s, nil
}

func (db Database) GetApiKey(workspaceID string, id uint) (*ApiKey, error) {
	var s ApiKey
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("id", id).
		Where("revoked", "false").
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &s, nil
}

func (db Database) AddApiKey(key *ApiKey) error {
	tx := db.Orm.Create(key)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) RevokeAPIKey(workspaceID string, id uint) error {
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("id", id).
		Updates(ApiKey{Revoked: true})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateActiveAPIKey(workspaceID string, id uint, value bool) error {
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("id", id).
		Update("active", value)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateAPIKeyRole(workspaceID string, id uint, role api.Role) error {
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("id", id).
		Update("role", role)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}
