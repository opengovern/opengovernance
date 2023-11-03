package db

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"gorm.io/gorm"
)

func (db Database) CreateRunnerJobs(runners []*model.ComplianceRunner) error {
	tx := db.ORM.
		Model(&model.ComplianceRunner{}).
		CreateInBatches(runners, 100)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) FetchCreatedRunners() ([]model.ComplianceRunner, error) {
	var jobs []model.ComplianceRunner
	tx := db.ORM.Model(&model.ComplianceRunner{}).
		Where("status = ?", runner.ComplianceRunnerCreated).Order("created_at ASC").Limit(100).Find(&jobs)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return jobs, nil
}

func (db Database) RetryFailedRunners() error {
	tx := db.ORM.Exec("UPDATE compliance_runners SET retry_count = retry_count + 1, status = 'CREATED' WHERE status = 'FAILED' AND retry_count < 3 AND updated_at < NOW() - interval '5 minutes'")
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateRunnerJob(
	id uint, status runner.ComplianceRunnerStatus, failureMsg string) error {
	tx := db.ORM.
		Model(&model.ComplianceRunner{}).
		Where("id = ?", id).
		Updates(model.ComplianceRunner{
			Status:         status,
			FailureMessage: failureMsg,
		})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateRunnerJobsTimedOut() error {
	tx := db.ORM.
		Model(&model.ComplianceRunner{}).
		Where("created_at < NOW() - INTERVAL '6 HOURS'").
		Where("status IN ?", []string{string(runner.ComplianceRunnerCreated), string(runner.ComplianceRunnerInProgress)}).
		Updates(model.ComplianceRunner{Status: runner.ComplianceRunnerFailed, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) ListRunnersWithID(ids []int64) ([]model.ComplianceRunner, error) {
	var jobs []model.ComplianceRunner
	tx := db.ORM.Where("id IN ?", ids).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return jobs, nil
}
