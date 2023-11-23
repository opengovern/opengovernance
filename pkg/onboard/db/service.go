package db

import (
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/db/model"
	"gorm.io/gorm"
)

type Database struct {
	Orm *gorm.DB
}

func NewDatabase(orm *gorm.DB) Database {
	return Database{
		Orm: orm,
	}
}

func (db Database) Initialize() error {
	err := db.Orm.AutoMigrate(
		&model.Connector{},
		&model.Credential{},
		&model.Source{},
		&model.ConnectionGroup{},
	)
	if err != nil {
		return err
	}

	return nil
}
