package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/analytics/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"gorm.io/gorm"
)

func (db Database) CountAnalyticsJobsByDate(start time.Time, end time.Time) (int64, error) {
	var count int64
	tx := db.ORM.Model(&model.AnalyticsJob{}).
		Where("status = ? AND updated_at >= ? AND updated_at < ?", api.JobCompleted, start, end).Count(&count)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, tx.Error
	}
	return count, nil
}

func (db Database) GetAnalyticsJobByID(jobID uint) (*model.AnalyticsJob, error) {
	var job model.AnalyticsJob
	tx := db.ORM.Model(&model.AnalyticsJob{}).Where("id = ?", jobID).First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &job, nil
}

func (db Database) ListAnalyticsJobs() ([]model.AnalyticsJob, error) {
	var jobs []model.AnalyticsJob
	tx := db.ORM.Model(&model.AnalyticsJob{}).Find(&jobs)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return jobs, nil
}

func (db Database) ListAnalyticsJobsByIds(ids []string) ([]model.AnalyticsJob, error) {
	var jobs []model.AnalyticsJob
	tx := db.ORM.Model(&model.AnalyticsJob{})

	if len(ids) > 0 {
		tx = tx.Where("id IN ?", ids)
	}

	tx = tx.Find(&jobs)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return jobs, nil
}

func (db Database) ListAnalyticsJobsByFilter(jobType []string, status []string, startTime *time.Time, endTime *time.Time) ([]model.AnalyticsJob, error) {
	var jobs []model.AnalyticsJob
	tx := db.ORM.Model(&model.AnalyticsJob{})

	if len(jobType) > 0 {
		tx = tx.Where("type IN (?)", jobType)
	}
	if len(status) > 0 {
		tx = tx.Where("status IN (?)", status)
	}
	if startTime != nil {
		tx = tx.Where("updated_at >= ?", startTime)
	}
	if endTime != nil {
		tx = tx.Where("updated_at <= ?", *endTime)
	}

	tx = tx.Find(&jobs)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return jobs, nil
}

func (db Database) FetchLastAnalyticsJobForJobType(analyticsJobType model.AnalyticsJobType) (*model.AnalyticsJob, error) {
	var job model.AnalyticsJob
	tx := db.ORM.Model(&model.AnalyticsJob{}).Order("created_at DESC").Where("type = ?", analyticsJobType).First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &job, nil
}

func (db Database) AddAnalyticsJob(job *model.AnalyticsJob) error {
	tx := db.ORM.Model(&model.AnalyticsJob{}).
		Create(job)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateAnalyticsJobStatus(job model.AnalyticsJob) error {
	tx := db.ORM.Model(&model.AnalyticsJob{}).
		Where("id = ?", job.ID).
		Update("status", job.Status)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateAnalyticsJobsTimedOut() error {
	tx := db.ORM.
		Model(&model.AnalyticsJob{}).
		Where(fmt.Sprintf("created_at < NOW() - INTERVAL '%d HOURS'", 2)).
		Where("status IN ?", []string{string(api.JobCreated), string(api.JobInProgress)}).
		Updates(model.AnalyticsJob{Status: api.JobCompletedWithFailure, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateAnalyticsJob(jobID uint, status api.JobStatus, failedMessage string) error {
	tx := db.ORM.Model(&model.AnalyticsJob{}).
		Where("id = ?", jobID).
		Updates(model.AnalyticsJob{
			Status:         status,
			FailureMessage: failedMessage,
		})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}
