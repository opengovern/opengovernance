package db

import (
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	insightapi "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	"gorm.io/gorm"
	"time"
)

func (db Database) CountInsightJobsByDate(start time.Time, end time.Time) (int64, error) {
	var count int64
	tx := db.ORM.Model(&model.InsightJob{}).
		Where("status = ? AND updated_at >= ? AND updated_at < ?", insightapi.InsightJobSucceeded, start, end).Count(&count)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, tx.Error
	}
	return count, nil
}

func (db Database) CleanupInsightJobsOlderThan(t time.Time) error {
	tx := db.ORM.Where("created_at < ?", t).Unscoped().Delete(&model.InsightJob{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) AddInsightJob(job *model.InsightJob) error {
	tx := db.ORM.Model(&model.InsightJob{}).
		Create(job)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateInsightJobStatus(job model.InsightJob) error {
	tx := db.ORM.Model(&model.InsightJob{}).
		Where("id = ?", job.ID).
		Update("status", job.Status)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateInsightJob(jobID uint, status insightapi.InsightJobStatus, failedMessage string) error {
	tx := db.ORM.Model(&model.InsightJob{}).
		Where("id = ?", jobID).
		Updates(model.InsightJob{
			Status:         status,
			FailureMessage: failedMessage,
		})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) FetchLastInsightJob() (*model.InsightJob, error) {
	var job model.InsightJob
	tx := db.ORM.Model(&model.InsightJob{}).
		Order("created_at DESC").First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &job, nil
}

func (db Database) GetLastInsightJobForResourceCollection(insightID uint, sourceID string,
	resourceCollectionId *string) (*model.InsightJob, error) {
	var job model.InsightJob
	tx := db.ORM.Model(&model.InsightJob{}).
		Where("source_id = ? AND insight_id = ?", sourceID, insightID).
		Order("created_at DESC")
	if resourceCollectionId == nil {
		tx = tx.Where("resource_collection IS NULL").First(&job)
	} else {
		tx = tx.Where("resource_collection = ?", *resourceCollectionId).First(&job)
	}
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &job, nil
}

func (db Database) ListInsightJobs() ([]model.InsightJob, error) {
	var job []model.InsightJob
	tx := db.ORM.Model(&model.InsightJob{}).Find(&job)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return job, nil
}

func (db Database) GetOldCompletedInsightJob(insightID uint, nDaysBefore int) (*model.InsightJob, error) {
	var job *model.InsightJob
	tx := db.ORM.Model(&model.InsightJob{}).
		Where("status = ?", insightapi.InsightJobSucceeded).
		Where("insight_id = ?", insightID).
		Where(fmt.Sprintf("updated_at <= now() - interval '%d days'", nDaysBefore)).
		Order("updated_at DESC").
		First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	} else if tx.RowsAffected != 1 {
		return nil, nil
	}
	return job, nil
}

func (db Database) GetInsightJobById(jobId uint) (*model.InsightJob, error) {
	var job model.InsightJob
	tx := db.ORM.Model(&model.InsightJob{}).
		Where("id = ?", jobId).
		Find(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &job, nil
}

func (db Database) GetInsightJobByInsightId(insightID uint) ([]model.InsightJob, error) {
	var jobs []model.InsightJob
	tx := db.ORM.Model(&model.InsightJob{}).
		Where("insight_id = ?", insightID).
		Find(&jobs)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return jobs, nil
}

// UpdateInsightJobsTimedOut updates the status of InsightJobs
// that have timed out while in the status of 'IN_PROGRESS' for longer
// than 4 hours.
func (db Database) UpdateInsightJobsTimedOut(insightIntervalHours time.Duration) error {
	tx := db.ORM.
		Model(&model.InsightJob{}).
		Where(fmt.Sprintf("created_at < NOW() - INTERVAL '%d HOURS'", int(insightIntervalHours.Hours()*2))).
		Where("status IN ?", []string{string(insightapi.InsightJobCreated)}).
		Updates(model.InsightJob{Status: insightapi.InsightJobFailed, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	tx = db.ORM.
		Model(&model.InsightJob{}).
		Where(fmt.Sprintf("created_at < NOW() - INTERVAL '30 MINUTES'")).
		Where("status IN ?", []string{string(insightapi.InsightJobInProgress)}).
		Updates(model.InsightJob{Status: insightapi.InsightJobFailed, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
