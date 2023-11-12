package db

import (
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"gorm.io/gorm"
)

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

func (db Database) UpdateAnalyticsJobsTimedOut(analyticsIntervalHours int64) error {
	tx := db.ORM.
		Model(&model.AnalyticsJob{}).
		Where(fmt.Sprintf("created_at < NOW() - INTERVAL '%d HOURS'", analyticsIntervalHours*2)).
		Where("status IN ?", []string{string(analytics.JobInProgress)}).
		Updates(model.AnalyticsJob{Status: analytics.JobCompletedWithFailure, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateAnalyticsJob(jobID uint, status analytics.JobStatus, failedMessage string) error {
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
