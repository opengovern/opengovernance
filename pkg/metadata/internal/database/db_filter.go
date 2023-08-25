package database

import (
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	"gorm.io/gorm"
)

type DatabaseFilter struct {
	orm *gorm.DB
}

func NewDatabaseFilter(orm *gorm.DB) Database {
	return Database{orm: orm}
}

func (db Database) InitializeFilter() error {
	err := db.orm.AutoMigrate(
		&models.ConfigMetadata{},
	)
	if err != nil {
		return err
	}

	return nil
}
