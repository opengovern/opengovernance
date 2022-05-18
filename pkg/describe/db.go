package describe

import (
	"fmt"
	"time"

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/api"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Database struct {
	orm *gorm.DB
}

func (db Database) Initialize() error {
	return db.orm.AutoMigrate(&Source{}, &DescribeSourceJob{}, &DescribeResourceJob{}, &ComplianceReportJob{})
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

// ListSources lists all sources
func (db Database) ListSources() ([]Source, error) {
	var sources []Source
	tx := db.orm.Find(&sources)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return sources, nil
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
		Where("id = ?", a.ID.String()).
		Delete(&Source{})
	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("delete source: didn't find the source to delete")
	}

	return nil
}

// GetSourceByUUID find source by uuid
func (db Database) GetSourceByUUID(id uuid.UUID) (*Source, error) {
	var source Source
	tx := db.orm.
		Where("id = ?", id.String()).
		Find(&source)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &source, nil
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
func (db Database) UpdateSourceDescribed(id uuid.UUID, describedAt time.Time) error {
	tx := db.orm.
		Model(&Source{}).
		Where("id = ?", id.String()).
		Updates(map[string]interface{}{
			"last_described_at": describedAt,                    // gorm.Expr("NOW()"),
			"next_describe_at":  describedAt.Add(2 * time.Hour), //gorm.Expr("NOW() + INTERVAL '2 HOURS'"),
		})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateSourceNextDescribeAtToNow updates the source next_describe_at to
// **NOW()**.
func (db Database) UpdateSourceNextDescribeAtToNow(id uuid.UUID) error {
	tx := db.orm.
		Model(&Source{}).
		Where("id = ?", id.String()).
		Where("next_describe_at > NOW() + interval '1 minute'").
		Updates(map[string]interface{}{
			"next_describe_at": gorm.Expr("NOW()"),
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
			"next_compliance_report_id": gorm.Expr("next_compliance_report_id + 1"),
		})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateSourceNextComplianceReportToNow updates the source next_compliance_report_at to
// **NOW()**.
func (db Database) UpdateSourceNextComplianceReportToNow(id uuid.UUID) error {
	tx := db.orm.
		Model(&Source{}).
		Where("id = ?", id.String()).
		Where("next_describe_at > NOW() + interval '1 minute'").
		Updates(map[string]interface{}{
			"next_compliance_report_at": gorm.Expr("NOW()"),
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
func (db Database) UpdateDescribeSourceJob(id uint, status api.DescribeSourceJobStatus) error {
	tx := db.orm.
		Model(&DescribeSourceJob{}).
		Where("id = ?", id).
		Updates(DescribeSourceJob{Status: status})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// ListAllDescribeSourceJobs lists all DescribeSourceJob .
func (db Database) ListAllDescribeSourceJobs() ([]DescribeSourceJob, error) {
	var jobs []DescribeSourceJob
	tx := db.orm.Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return jobs, nil
}

// ListDescribeSourceJobs lists the DescribeSourceJobs for the given sourcel.
func (db Database) ListDescribeSourceJobs(sourceID uuid.UUID) ([]DescribeSourceJob, error) {
	var jobs []DescribeSourceJob
	tx := db.orm.Preload(clause.Associations).Where("source_id = ?", sourceID).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return jobs, nil
}

type DescribedSourceJobDescribeResourceJobStatus struct {
	DescribeSourceJobID       uint                          `gorm:"column:id"`
	DescribeResourceJobStatus api.DescribeResourceJobStatus `gorm:"column:status"`
	DescribeResourceJobCount  int                           `gorm:"column:count"`
}

// Finds the DescribeSourceJobs that are IN_PROGRESS and find the
// status of the corresponding DescribeResourceJobs and their counts.
func (db Database) QueryInProgressDescribedSourceJobGroupByDescribeResourceJobStatus() ([]DescribedSourceJobDescribeResourceJobStatus, error) {
	var results []DescribedSourceJobDescribeResourceJobStatus

	tx := db.orm.
		Model(&DescribeSourceJob{}).
		Select("describe_source_jobs.id, describe_resource_jobs.status, COUNT(*)").
		Joins("JOIN describe_resource_jobs ON describe_source_jobs.id = describe_resource_jobs.parent_job_id").
		Where("describe_source_jobs.status IN ?", []string{string(api.DescribeSourceJobInProgress)}).
		Group("describe_source_jobs.id").
		Group("describe_resource_jobs.status").
		Order("describe_source_jobs.id ASC").
		Find(&results)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return results, nil
}

func (db Database) QueryOlderThanNRecentCompletedDescribeSourceJobs(n int) ([]DescribeSourceJob, error) {
	var results []DescribeSourceJob

	tx := db.orm.Raw(
		`
SELECT jobs.id
FROM (
	SELECT *, rank() OVER ( 
		PARTITION BY source_id 
		ORDER BY updated_at DESC
	)
	FROM describe_source_jobs 
	WHERE status IN ? AND deleted_at IS NULL) 
jobs
WHERE rank > ?
`, []string{string(api.DescribeSourceJobCompleted), string(api.DescribeSourceJobCompletedWithFailure)}, n).Scan(&results)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return results, nil
}

func (db Database) DeleteDescribeSourceJob(id uint) error {
	tx := db.orm.
		Where("id = ?", id).
		Delete(&DescribeSourceJob{})
	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("delete source: didn't find the describe source job to delete")
	}

	return nil
}

// =============================== DescribeResourceJob ===============================

func (db Database) GetDescribeResourceJob(id uint) (DescribeResourceJob, error) {
	var job DescribeResourceJob
	tx := db.orm.Where("id = ?", id).First(&job)
	if tx.Error != nil {
		return DescribeResourceJob{}, tx.Error
	}

	return job, nil
}

// UpdateDescribeResourceJobStatus updates the status of the DescribeResourceJob to the provided status.
// If the status if 'FAILED', msg could be used to indicate the failure reason
func (db Database) UpdateDescribeResourceJobStatus(id uint, status api.DescribeResourceJobStatus, msg string) error {
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
		Where("status IN ?", []string{string(api.DescribeResourceJobCreated), string(api.DescribeResourceJobQueued)}).
		Updates(DescribeResourceJob{Status: api.DescribeResourceJobFailed, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// ListAllDescribeResourceJobs lists all the DescribeResourceJobs.
func (db Database) ListAllDescribeResourceJobs() ([]DescribeResourceJob, error) {
	var jobs []DescribeResourceJob
	tx := db.orm.Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return jobs, nil
}

// ListDescribeResourceJobs lists the DescribeResourceJob for the given source job .
func (db Database) ListDescribeResourceJobs(describeSourceJobID uint) ([]DescribeResourceJob, error) {
	var jobs []DescribeResourceJob
	tx := db.orm.Where("parent_job_id = ?", describeSourceJobID).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return jobs, nil
}

func (db Database) DeleteDescribeResourceJob(id uint) error {
	tx := db.orm.
		Where("id = ?", id).
		Delete(&DescribeResourceJob{})
	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("delete source: didn't find the describe resource job to delete")
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
	id uint, status api2.ComplianceReportJobStatus, reportCreatedAt int64, failureMsg string) error {
	tx := db.orm.
		Model(&ComplianceReportJob{}).
		Where("id = ?", id).
		Updates(ComplianceReportJob{
			Status:          status,
			ReportCreatedAt: reportCreatedAt,
			FailureMessage:  failureMsg,
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
		Where("status IN ?", []string{string(api2.ComplianceReportJobCreated), string(api2.ComplianceReportJobInProgress)}).
		Updates(ComplianceReportJob{Status: api2.ComplianceReportJobCompletedWithFailure, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// ListComplianceReports lists the ComplianceReportJob .
func (db Database) ListComplianceReports(sourceID uuid.UUID) ([]ComplianceReportJob, error) {
	var jobs []ComplianceReportJob
	tx := db.orm.Where("source_id = ?", sourceID).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return jobs, nil
}

// GetLastCompletedSourceComplianceReport returns the ComplianceReportJob which is completed.
func (db Database) GetLastCompletedSourceComplianceReport(sourceID uuid.UUID) (*ComplianceReportJob, error) {
	var job ComplianceReportJob
	tx := db.orm.
		Where("source_id = ? AND status = ?", sourceID, api2.ComplianceReportJobCompleted).
		Order("updated_at DESC").
		First(&job)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &job, nil
}

// GetLastCompletedComplianceReportID returns id of last completed compliance report.
func (db Database) GetLastCompletedComplianceReportID() (uint, error) {
	var id uint
	tx := db.orm.
		Select("MIN(next_compliance_report_id)").
		First(&id)
	if tx.Error != nil {
		return 0, tx.Error
	}

	return id - 1, nil
}

// ListCompletedComplianceReportByDate returns list of ComplianceReportJob s which is completed within the date range.
func (db Database) ListCompletedComplianceReportByDate(sourceID uuid.UUID, fromDate, toDate time.Time) ([]ComplianceReportJob, error) {
	var jobs []ComplianceReportJob
	tx := db.orm.
		Where("source_id = ? AND status = ? AND updated_at > ? AND updated_at < ?",
			sourceID, api2.ComplianceReportJobCompleted, fromDate, toDate).
		Order("updated_at DESC").
		Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return jobs, nil
}

func (db Database) DeleteComplianceReportJob(id uint) error {
	tx := db.orm.
		Where("id = ?", id).
		Delete(&ComplianceReportJob{})
	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("delete compliance report: didn't find the compliance report job to delete")
	}

	return nil
}

func (db Database) QueryOlderThanNRecentCompletedComplianceReportJobs(n int) ([]ComplianceReportJob, error) {
	var results []ComplianceReportJob
	tx := db.orm.Raw(
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
`, []string{string(api.DescribeSourceJobCompleted), string(api.DescribeSourceJobCompletedWithFailure)}, n).Scan(&results)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return results, nil
}
