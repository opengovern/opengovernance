package db

import (
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"gorm.io/gorm"
	"time"
)

func (db Database) CountComplianceJobsByDate(start time.Time, end time.Time) (int64, error) {
	var count int64
	tx := db.ORM.Model(&model.ComplianceJob{}).
		Where("status = ? AND updated_at >= ? AND updated_at < ?", model.ComplianceJobSucceeded, start, end).Count(&count)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, tx.Error
	}
	return count, nil
}

func (db Database) CreateComplianceJob(job *model.ComplianceJob) error {
	tx := db.ORM.
		Model(&model.ComplianceJob{}).
		Create(job)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateComplianceJob(
	id uint, status model.ComplianceJobStatus, failureMsg string) error {
	tx := db.ORM.
		Model(&model.ComplianceJob{}).
		Where("id = ?", id).
		Updates(model.ComplianceJob{
			Status:         status,
			FailureMessage: failureMsg,
		})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateComplianceJobsTimedOut(complianceIntervalHours int64) error {
	tx := db.ORM.
		Model(&model.ComplianceJob{}).
		Where(fmt.Sprintf("created_at < NOW() - INTERVAL '%d HOURS'", complianceIntervalHours)).
		Where("status IN ?", []string{string(model.ComplianceJobCreated),
			string(model.ComplianceJobRunnersInProgress),
			string(model.ComplianceJobSummarizerInProgress),
		}).
		Updates(model.ComplianceJob{Status: model.ComplianceJobFailed, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) GetComplianceJobByID(ID uint) (*model.ComplianceJob, error) {
	var job model.ComplianceJob
	tx := db.ORM.Where("id = ?", ID).Find(&job)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &job, nil
}

func (db Database) CleanupComplianceJobsOlderThan(t time.Time) error {
	tx := db.ORM.Where("updated_at < ?", t).Unscoped().Delete(&model.ComplianceJob{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) GetLastComplianceJob(benchmarkID string) (*model.ComplianceJob, error) {
	var job model.ComplianceJob
	tx := db.ORM.Model(&model.ComplianceJob{}).Where("benchmark_id = ?", benchmarkID).Order("created_at DESC").First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &job, nil
}

func (db Database) ListComplianceJobs() ([]model.ComplianceJob, error) {
	var job []model.ComplianceJob
	tx := db.ORM.Model(&model.ComplianceJob{}).First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return job, nil
}

func (db Database) ListComplianceRunnersWithStatus(status model.ComplianceJobStatus) ([]model.ComplianceJob, error) {
	var jobs []model.ComplianceJob
	tx := db.ORM.Where("status = ?", status).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return jobs, nil
}

func (db Database) SetJobToRunnersInProgress() error {
	tx := db.ORM.Exec(`
UPDATE compliance_jobs j SET status = 'RUNNERS_IN_PROGRESS' WHERE status = 'CREATED' AND
	(select count(*) from compliance_runners where parent_job_id = j.id) > 0
`)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) ListJobsWithRunnersCompleted() ([]model.ComplianceJob, error) {
	var jobs []model.ComplianceJob
	tx := db.ORM.Raw(`
SELECT * FROM compliance_jobs j WHERE status = 'RUNNERS_IN_PROGRESS' AND
	(select count(*) from compliance_runners where parent_job_id = j.id AND 
	                                               NOT (status = 'SUCCEEDED' OR status = 'TIMEOUT' OR (status = 'FAILED' and retry_count >= 3))
	                                         ) = 0
`).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return jobs, nil
}

func (db Database) GetLastUpdatedRunnerForParent(jobId uint) (model.ComplianceRunner, error) {
	var runner model.ComplianceRunner
	tx := db.ORM.Where("parent_job_id = ?", jobId).Order("updated_at DESC").First(&runner)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return model.ComplianceRunner{}, nil
		}
		return model.ComplianceRunner{}, tx.Error
	}

	return runner, nil
}

func (db Database) FetchTotalFindingCountForComplianceJob(jobID uint) (int, error) {
	var count int
	tx := db.ORM.Raw(`SELECT sum(coalesce(total_finding_count,0)) FROM compliance_runners WHERE parent_job_id = ?`, jobID).Scan(&count)
	if tx.Error != nil {
		return 0, tx.Error
	}

	return count, nil
}

func (db Database) ListJobsToFinish() ([]model.ComplianceJob, error) {
	var jobs []model.ComplianceJob
	tx := db.ORM.Raw(`
SELECT * FROM compliance_jobs j WHERE status = 'SUMMARIZER_IN_PROGRESS' AND
	(select count(*) from compliance_summarizers where parent_job_id = j.id AND (status = 'SUCCEEDED' OR (status = 'FAILED' and retry_count >= 3))) > 0
`).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return jobs, nil
}
