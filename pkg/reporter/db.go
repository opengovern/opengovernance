package reporter

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Database struct {
	orm *gorm.DB
}

func NewDatabase(orm *gorm.DB) (*Database, error) {
	db := &Database{orm: orm}

	err := db.orm.AutoMigrate(
		DatabaseWorkerJob{},
		WorkerJobResult{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (db Database) Close() error {
	sqlDB, err := db.orm.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

func (db Database) BatchInsertWorkerJobResults(results []WorkerJobResult) error {
	return db.orm.Create(&results).Error
}

func (db Database) InsertWorkerJobResult(result *WorkerJobResult) error {
	return db.orm.Create(result).Error
}

func (db Database) InsertWorkerJob(job *DatabaseWorkerJob) error {
	return db.orm.Create(job).Error
}

func (db Database) GetWorkerJob(jobID uint) (DatabaseWorkerJob, error) {
	var job DatabaseWorkerJob
	err := db.orm.Preload(clause.Associations).First(&job, jobID).Error
	return job, err
}

func (db Database) UpdateWorkerJobStatus(jobID int, status JobStatus) error {
	return db.orm.Model(&DatabaseWorkerJob{}).Where("id = ?", jobID).Update("status", status).Error
}
