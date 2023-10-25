package db

import (
	"fmt"
	"github.com/google/uuid"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"time"
)

// =============================== model.ComplianceJob ===============================

// CreateComplianceJob creates a new ComplianceJob.
// If there is no error, the job is updated with the assigned ID
func (db Database) CreateComplianceJob(job *model.ComplianceJob) error {
	tx := db.ORM.
		Model(&model.ComplianceJob{}).
		Create(job)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateComplianceJob updates the model.ComplianceJob
func (db Database) UpdateComplianceJob(
	id uint, status api2.ComplianceReportJobStatus, failureMsg string) error {
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

// UpdateComplianceJobsTimedOut updates the status of ComplianceJob
// that have timed out while in the status of 'CREATED' or 'QUEUED' for longer
// than 4 hours.
func (db Database) UpdateComplianceJobsTimedOut(complianceIntervalHours int64) error {
	tx := db.ORM.
		Model(&model.ComplianceJob{}).
		Where(fmt.Sprintf("created_at < NOW() - INTERVAL '%d HOURS'", complianceIntervalHours)).
		Where("status IN ?", []string{string(api2.ComplianceReportJobCreated), string(api2.ComplianceReportJobInProgress)}).
		Updates(model.ComplianceJob{Status: api2.ComplianceReportJobCompletedWithFailure, FailureMessage: "Job timed out"})
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

// GetLastCompletedComplianceReportID returns id of last completed compliance report.
func (db Database) GetLastCompletedComplianceReportID() (uint, error) {
	var id uint
	tx := db.ORM.
		Select("MIN(next_compliance_report_id)").
		First(&id)
	if tx.Error != nil {
		return 0, tx.Error
	}

	return id - 1, nil
}

// ListCompletedComplianceReportByDate returns list of model.ComplianceJob s which is completed within the date range.
func (db Database) ListCompletedComplianceReportByDate(sourceID uuid.UUID, fromDate, toDate time.Time) ([]model.ComplianceJob, error) {
	var jobs []model.ComplianceJob
	tx := db.ORM.
		Where("source_id = ? AND status = ? AND updated_at > ? AND updated_at < ?",
			sourceID, api2.ComplianceReportJobCompleted, fromDate, toDate).
		Order("updated_at DESC").
		Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return jobs, nil
}

func (db Database) ListComplianceReportsWithFilter(
	timeAfter, timeBefore *time.Time,
	connectionID *string, connector *source.Type,
	benchmarkID *string, resourceCollection **string) ([]model.ComplianceJob, error) {
	var jobs []model.ComplianceJob
	tx := db.ORM
	if timeAfter != nil {
		tx = tx.Where("created_at >= ?", *timeAfter)
	}
	if timeBefore != nil {
		tx = tx.Where("created_at <= ?", *timeBefore)
	}
	if connectionID != nil {
		tx = tx.Where("source_id = ?", *connectionID)
	}
	if connector != nil {
		tx = tx.Where("source_type = ?", connector.String())
	}
	if benchmarkID != nil {
		tx = tx.Where("benchmark_id = ?", benchmarkID)
	}
	if resourceCollection != nil {
		rc := *resourceCollection
		if rc != nil {
			tx = tx.Where("resource_collection = ?", *rc)
		} else {
			tx = tx.Where("resource_collection IS NULL")
		}
	}
	tx = tx.Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return jobs, nil
}

func (db Database) CleanupComplianceJobsOlderThan(t time.Time) error {
	tx := db.ORM.Where("updated_at < ?", t).Unscoped().Delete(&model.ComplianceJob{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) DeleteComplianceJob(id uint) error {
	tx := db.ORM.
		Where("id = ?", id).
		Unscoped().
		Delete(&model.ComplianceJob{})
	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("delete compliance report: didn't find the compliance report job to delete")
	}

	return nil
}

func (db Database) QueryOlderThanNRecentCompletedComplianceJobs(n int) ([]model.ComplianceJob, error) {
	var results []model.ComplianceJob
	tx := db.ORM.Raw(
		`
SELECT jobs.id
FROM (
	SELECT *, rank() OVER ( 
		PARTITION BY source_id 
		ORDER BY updated_at DESC
	)
	FROM compliance_report_jobs 
	WHERE status IN ? AND deleted_at IS NULL) 
jobs
WHERE rank > ?
`, []string{string(api2.ComplianceReportJobCompleted), string(api2.ComplianceReportJobCompletedWithFailure)}, n).Scan(&results)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return results, nil
}

func (db Database) QueryComplianceJobs(id string) ([]model.ComplianceJob, error) {
	status := []string{string(api2.ComplianceReportJobCompleted), string(api2.ComplianceReportJobCompletedWithFailure)}

	var jobs []model.ComplianceJob
	tx := db.ORM.Where("status IN ? AND deleted_at IS NULL AND source_id = ?", status, id).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return jobs, nil
}
