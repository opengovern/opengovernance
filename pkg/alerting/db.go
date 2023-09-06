package alerting

import "gorm.io/gorm"

type Database struct {
	orm *gorm.DB
}

func NewDatabase(orm *gorm.DB) Database {
	return Database{orm: orm}
}

func (db Database) Initialize() error {
	err := db.orm.AutoMigrate(
		&ActionType{},
		&RuleType{},
	)
	if err != nil {
		return err
	}

	return nil
}
