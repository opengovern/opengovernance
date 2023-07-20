package db

import "gorm.io/gorm"

type Database struct {
	orm *gorm.DB
}

func NewDatabase(orm *gorm.DB) Database {
	return Database{orm: orm}
}

func (db Database) Initialize() error {
	err := db.orm.AutoMigrate(
		&Metric{},
	)
	if err != nil {
		return err
	}

	return nil
}
