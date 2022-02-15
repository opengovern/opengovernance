package describe

import (
	"fmt"
	compliance_report "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Database struct {
	orm *gorm.DB
}

// ==================================== Source ====================================

// CreateSource creates an source.
func (db Database) CreateSource(a *Source) error {
	tx := db.orm.
		Model(&Source{}).
		Clauses(clause.OnConflict{DoNothing: true}). // Don't update an existing source
		Create(a)
	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("create source: didn't create source due to id conflict")
	}

	return nil
}

// UpdateSource updates the source information.
func (db Database) UpdateSource(a *Source) error {
	tx := db.orm.
		Model(&Source{}).
		Where("id = ?", a.ID.String()).
		Updates(a)
	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("update source: didn't find the source to update")
	}

	return nil
}

// DeleteSource deletes the source
func (db Database) DeleteSource(a Source) error {
	tx := db.orm.
		Delete(&Source{}, a.ID.String())
	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("delete source: didn't find the source to delete")
	}

	return nil
}

// CreateSources creates multiple source in batches.
func (db Database) CreateSources(a []Source) error {
	tx := db.orm.
		Model(&Source{}).
		Create(a)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateSourceDescribed updates the source last_described_at to
// **NOW()** and next_describe_at to **NOW() + 2 Hours**.
func (db Database) UpdateSourceDescribed(id uuid.UUID) error {
	tx := db.orm.
		Model(&Source{}).
		Where("id = ?", id.String()).
		Updates(map[string]interface{}{
			"last_described_at": gorm.Expr("NOW()"),
			"next_describe_at":  gorm.Expr("NOW() + INTERVAL '2 HOURS'"),
		})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateSourceReportGenerated updates the source last_compliance_report_at to
// **NOW()** and next_compliance_report_at to **NOW() + 2 Hours**.
func (db Database) UpdateSourceReportGenerated(id uuid.UUID) error {
	tx := db.orm.
		Model(&Source{}).
		Where("id = ?", id.String()).
		Updates(map[string]interface{}{
			"last_compliance_report_at": gorm.Expr("NOW()"),
			"next_compliance_report_at": gorm.Expr("NOW() + INTERVAL '2 HOURS'"),
		})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// QuerySourcesDueForDescribe queries for all the sources that
// are due for another describe.
func (db Database) QuerySourcesDueForDescribe() ([]Source, error) {
	var sources []Source
	tx := db.orm.
		Where("next_describe_at < NOW()").
		Find(&sources)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return sources, nil
}

// QuerySourcesDueForComplianceReport queries for all the sources that
// are due for another steampipe check.
func (db Database) QuerySourcesDueForComplianceReport() ([]Source, error) {
	var sources []Source
	tx := db.orm.
		Where("next_compliance_report_at < NOW()").
		Find(&sources)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return sources, nil
}

// =============================== DescribeSourceJob ===============================

// CreateDescribeSourceJob creates a new DescribeSourceJob.
// If there is no error, the job is updated with the assigned ID
func (db Database) CreateDescribeSourceJob(job *DescribeSourceJob) error {
	tx := db.orm.
		Model(&DescribeSourceJob{}).
		Create(job)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateDescribeSourceJob updates the DescribeSourceJob status.
func (db Database) UpdateDescribeSourceJob(id uint, status DescribeSourceJobStatus) error {
	tx := db.orm.
		Model(&DescribeSourceJob{}).
		Where("id = ?", id).
		Updates(DescribeSourceJob{Status: status})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

type DescribedSourceJobDescribeResourceJobStatus struct {
	DescribeSourceJobID       uint                      `gorm:"column:id"`
	DescribeResourceJobStatus DescribeResourceJobStatus `gorm:"column:status"`
	DescribeResourceJobCount  int                       `gorm:"column:count"`
}

// Finds the DescribeSourceJobs that are IN_PROGRESS and find the
// status of the corresponding DescribeResourceJobs and their counts.
func (db Database) QueryInProgressDescribedSourceJobGroupByDescribeResourceJobStatus() ([]DescribedSourceJobDescribeResourceJobStatus, error) {
	var results []DescribedSourceJobDescribeResourceJobStatus

	tx := db.orm.
		Model(&DescribeSourceJob{}).
		Select("describe_source_jobs.id, describe_resource_jobs.status, COUNT(*)").
		Joins("JOIN describe_resource_jobs ON describe_source_jobs.id = describe_resource_jobs.parent_job_id").
		Where("describe_source_jobs.status IN ?", []string{string(DescribeSourceJobInProgress)}).
		Group("describe_source_jobs.id").
		Group("describe_resource_jobs.status").
		Order("describe_source_jobs.id ASC").
		Find(&results)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return results, nil
}

// =============================== DescribeResourceJob ===============================

// UpdateDescribeResourceJobStatus updates the status of the DescribeResourceJob to the provided status.
// If the status if 'FAILED', msg could be used to indicate the failure reason
func (db Database) UpdateDescribeResourceJobStatus(id uint, status DescribeResourceJobStatus, msg string) error {
	tx := db.orm.
		Model(&DescribeResourceJob{}).
		Where("id = ?", id).
		Updates(DescribeResourceJob{Status: status, FailureMessage: msg})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateDescribeResourceJobsTimedOut updates the status of DescribeResourceJobs
// that have timed out while in the status of 'CREATED' or 'QUEUED' for longer
// than 4 hours.
func (db Database) UpdateDescribeResourceJobsTimedOut() error {
	tx := db.orm.
		Model(&DescribeResourceJob{}).
		Where("created_at < NOW() - INTERVAL '4 HOURS'").
		Where("status IN ?", []string{string(DescribeResourceJobCreated), string(DescribeResourceJobQueued)}).
		Updates(DescribeResourceJob{Status: DescribeResourceJobFailed, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// =============================== ComplianceReportJob ===============================

// CreateComplianceReportJob creates a new ComplianceReportJob.
// If there is no error, the job is updated with the assigned ID
func (db Database) CreateComplianceReportJob(job *ComplianceReportJob) error {
	tx := db.orm.
		Model(&ComplianceReportJob{}).
		Create(job)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateComplianceReportJob updates the ComplianceReportJob
func (db Database) UpdateComplianceReportJob(
	id uint, status compliance_report.ComplianceReportJobStatus, failureMsg string, s3ResultURL string) error {
	tx := db.orm.
		Model(&ComplianceReportJob{}).
		Where("id = ?", id).
		Updates(ComplianceReportJob{
			Status: status,
			FailureMessage: failureMsg,
			S3ResultURL: s3ResultURL,
		})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateComplianceReportJobsTimedOut updates the status of ComplianceReportJob
// that have timed out while in the status of 'CREATED' or 'QUEUED' for longer
// than 4 hours.
func (db Database) UpdateComplianceReportJobsTimedOut() error {
	tx := db.orm.
		Model(&ComplianceReportJob{}).
		Where("created_at < NOW() - INTERVAL '4 HOURS'").
		Where("status IN ?", []string{string(compliance_report.ComplianceReportJobCreated), string(compliance_report.ComplianceReportJobInProgress)}).
		Updates(ComplianceReportJob{Status: compliance_report.ComplianceReportJobCompletedWithFailure, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
