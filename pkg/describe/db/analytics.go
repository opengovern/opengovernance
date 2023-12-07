package db

import (
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"gorm.io/gorm"
	"time"
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
	tx := db.ORM.Model(&model.AnalyticsJob{}).First(&jobs)
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

func (db Database) UpdateAnalyticsJobsTimedOut(analyticsIntervalHours time.Duration) error {
	tx := db.ORM.
		Model(&model.AnalyticsJob{}).
		Where(fmt.Sprintf("created_at < NOW() - INTERVAL '%d HOURS'", int(analyticsIntervalHours.Hours()*2))).
		Where("status IN ?", []string{string(api.JobInProgress)}).
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
