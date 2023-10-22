package db

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
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
		&StoreCostTableJob{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (db Database) CreateStoreCostTableJob() error {
	job := StoreCostTableJob{}
	return db.orm.Model(&StoreCostTableJob{}).Create(&job).Error
}

func (db Database) UpdateStoreCostTableJob(id string, status StoreCostTableJobStatus) error {
	return db.orm.Model(&StoreCostTableJob{}).Where("id = ?", id).Updates(StoreCostTableJob{Status: status}).Error
}

func (db Database) GetLastJob(connector source.Type) (StoreCostTableJob, error) {
	var job StoreCostTableJob
	err := db.orm.Model(&StoreCostTableJob{}).Where("connector = ?", connector).Order("UpdatedAt").First(&job).Error
	if err != nil {
		return StoreCostTableJob{}, err
	}
	return job, nil
}
