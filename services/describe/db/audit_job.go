package db

import (
	"errors"
	"github.com/opengovern/opencomply/services/describe/db/model"
	"gorm.io/gorm"
)

func (db Database) CreateComplianceQuickRun(job *model.ComplianceQuickRun) (uint, error) {
	tx := db.ORM.Model(&model.ComplianceQuickRun{}).
		Create(job)
	if tx.Error != nil {
		return 0, tx.Error
	}

	return job.ID, nil
}

func (db Database) GetComplianceQuickRunByID(ID uint) (*model.ComplianceQuickRun, error) {
	var job model.ComplianceQuickRun
	tx := db.ORM.Where("id = ?", ID).Find(&job)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &job, nil
}

func (db Database) ListComplianceQuickRuns() ([]model.ComplianceQuickRun, error) {
	var job []model.ComplianceQuickRun
	tx := db.ORM.Model(&model.ComplianceQuickRun{}).First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return job, nil
}

func (db Database) CleanupAllComplianceQuickRuns() error {
	tx := db.ORM.Where("1 = 1").Unscoped().Delete(&model.ComplianceQuickRun{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateTimedOutInProgressComplianceQuickRuns() error {
	tx := db.ORM.
		Model(&model.ComplianceQuickRun{}).
		Where("status = ?", model.ComplianceQuickRunStatusInProgress).
		Where("updated_at < NOW() - INTERVAL '10 MINUTES'").
		Updates(model.ComplianceQuickRun{Status: model.ComplianceQuickRunStatusTimeOut, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateTimedOutQueuedComplianceQuickRuns() error {
	tx := db.ORM.
		Model(&model.ComplianceQuickRun{}).
		Where("status = ?", model.ComplianceQuickRunStatusQueued).
		Where("updated_at < NOW() - INTERVAL '12 HOURS'").
		Updates(model.ComplianceQuickRun{Status: model.ComplianceQuickRunStatusTimeOut, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) FetchCreatedComplianceQuickRuns() ([]model.ComplianceQuickRun, error) {
	var jobs []model.ComplianceQuickRun
	tx := db.ORM.Model(&model.ComplianceQuickRun{}).Where("status = ?", model.ComplianceQuickRunStatusCreated).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return jobs, nil
}

func (db Database) UpdateComplianceQuickRunStatus(jobId uint, status model.ComplianceQuickRunStatus, failureReason string) error {
	tx := db.ORM.Model(&model.ComplianceQuickRun{}).Where("id = ?", jobId).
		Updates(model.ComplianceQuickRun{Status: status, FailureMessage: failureReason})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateComplianceQuickRunNatsSeqNum(
	id uint, seqNum uint64) error {
	tx := db.ORM.
		Model(&model.ComplianceQuickRun{}).
		Where("id = ?", id).
		Updates(model.ComplianceQuickRun{
			NatsSequenceNumber: seqNum,
		})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
