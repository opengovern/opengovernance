package database

import (
	"github.com/opengovern/opencomply/services/metadata/models"
	"gorm.io/gorm"
)

type Database struct {
	orm *gorm.DB
}

func NewDatabase(orm *gorm.DB) Database {
	return Database{orm: orm}
}

func (db Database) Initialize() error {
	err := db.orm.AutoMigrate(
		&models.ConfigMetadata{},
		&models.QueryParameterValues{},
		&models.QueryView{},
		&models.QueryViewTag{},
		&models.Query{},
		&models.Parameters{},
		&models.PlatformConfiguration{},
	)
	if err != nil {
		return err
	}

	return nil
}
