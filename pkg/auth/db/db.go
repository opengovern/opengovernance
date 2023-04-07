package db

import (
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

func (db Database) GetApiKeys(workspaceID string, id uint) (*ApiKey, error) {
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
