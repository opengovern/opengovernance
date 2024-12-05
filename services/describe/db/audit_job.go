package db

import (
	"errors"
	"github.com/opengovern/opencomply/services/describe/db/model"
	"gorm.io/gorm"
)

func (db Database) CreateAuditJob(job *model.AuditJob) (uint, error) {
	tx := db.ORM.Model(&model.AuditJob{}).
		Create(job)
	if tx.Error != nil {
		return 0, tx.Error
	}

	return job.ID, nil
}

func (db Database) GetAuditJobByID(ID uint) (*model.AuditJob, error) {
	var job model.AuditJob
	tx := db.ORM.Where("id = ?", ID).Find(&job)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &job, nil
}

func (db Database) ListAuditJobs() ([]model.AuditJob, error) {
	var job []model.AuditJob
	tx := db.ORM.Model(&model.AuditJob{}).First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return job, nil
}

func (db Database) CleanupAllAuditJobs() error {
	tx := db.ORM.Where("1 = 1").Unscoped().Delete(&model.AuditJob{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateTimedOutInProgressAuditJobs() error {
	tx := db.ORM.
		Model(&model.AuditJob{}).
		Where("status = ?", model.AuditJobStatusInProgress).
		Where("updated_at < NOW() - INTERVAL '10 MINUTES'").
		Updates(model.AuditJob{Status: model.AuditJobStatusTimeOut, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateTimedOutQueuedAuditJobs() error {
	tx := db.ORM.
		Model(&model.AuditJob{}).
		Where("status = ?", model.AuditJobStatusQueued).
		Where("updated_at < NOW() - INTERVAL '12 HOURS'").
		Updates(model.AuditJob{Status: model.AuditJobStatusTimeOut, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) FetchCreatedAuditJobs() ([]model.AuditJob, error) {
	var jobs []model.AuditJob
	tx := db.ORM.Model(&model.AuditJob{}).Where("status = ?", model.AuditJobStatusCreated).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return jobs, nil
}

func (db Database) UpdateAuditJobStatus(jobId uint, status model.AuditJobStatus, failureReason string) error {
	tx := db.ORM.Model(&model.AuditJob{}).Where("id = ?", jobId).
		Updates(model.AuditJob{Status: status, FailureMessage: failureReason})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateAuditJobNatsSeqNum(
	id uint, seqNum uint64) error {
	tx := db.ORM.
		Model(&model.AuditJob{}).
		Where("id = ?", id).
		Updates(model.AuditJob{
			NatsSequenceNumber: seqNum,
		})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
