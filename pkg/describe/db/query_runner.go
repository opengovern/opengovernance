package db

import (
	"github.com/kaytu-io/open-governance/pkg/describe/db/model"
	query_runner "github.com/kaytu-io/open-governance/pkg/inventory/query-runner"
)

func (db Database) CreateQueryRunnerJob(job model.QueryRunnerJob) error {
	tx := db.ORM.Model(&model.QueryRunnerJob{}).Create(job)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) GetQueryRunnerJob(id uint) (*model.QueryRunnerJob, error) {
	var job model.QueryRunnerJob
	tx := db.ORM.Model(&model.QueryRunnerJob{}).Where("run_id = ?", id).First(&job)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &job, nil
}

func (db Database) FetchCreatedQueryRunnerJobs() ([]model.QueryRunnerJob, error) {
	var jobs []model.QueryRunnerJob
	tx := db.ORM.Model(&model.QueryRunnerJob{}).Where("status = ?", query_runner.QueryRunnerCreated).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return jobs, nil
}

func (db Database) DeleteQueryRunnerJob(id uint) error {
	tx := db.ORM.Model(&model.QueryRunnerJob{}).Delete(&model.QueryRunnerJob{}, id)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateQueryRunnerJobStatus(jobId uint, status query_runner.QueryRunnerStatus) error {
	tx := db.ORM.Model(&model.QueryRunnerJob{}).Where("run_id = ?", jobId).Updates(model.QueryRunnerJob{Status: status})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}
