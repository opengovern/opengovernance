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

func (db Database) CreateStoreCostTableJob(connector source.Type) (uint, error) {
	job := StoreCostTableJob{Connector: connector, Status: StoreCostTableJobStatusProcessing}
	err := db.orm.Model(&StoreCostTableJob{}).Create(&job).Error
	if err != nil {
		return 0, err
	}
	return job.Id, nil
}

func (db Database) UpdateStoreCostTableJob(id uint, status StoreCostTableJobStatus, errorMessage string, count int64) error {
	return db.orm.Model(&StoreCostTableJob{}).Where("id = ?", id).
		Updates(StoreCostTableJob{Status: status, ErrorMessage: errorMessage, Count: count}).Error
}

func (db Database) GetLastJob(connector source.Type) (StoreCostTableJob, error) {
	var job StoreCostTableJob
	err := db.orm.Model(&StoreCostTableJob{}).Where("connector = ?", connector).Order("UpdatedAt").First(&job).Error
	if err != nil {
		return StoreCostTableJob{}, err
	}
	return job, nil
}
