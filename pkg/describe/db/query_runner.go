package db

import (
	"github.com/kaytu-io/open-governance/pkg/compliance/runner"
	"github.com/kaytu-io/open-governance/pkg/describe/db/model"
	queryrunner "github.com/kaytu-io/open-governance/pkg/inventory/query-runner"
)

func (db Database) CreateQueryRunnerJob(job *model.QueryRunnerJob) (uint, error) {
	tx := db.ORM.Create(job)
	if tx.Error != nil {
		return 0, tx.Error
	}

	return job.ID, nil
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
	tx := db.ORM.Model(&model.QueryRunnerJob{}).Where("status = ?", queryrunner.QueryRunnerCreated).Find(&jobs)
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

func (db Database) UpdateQueryRunnerJobStatus(jobId uint, status queryrunner.QueryRunnerStatus, failureReason string) error {
	tx := db.ORM.Model(&model.QueryRunnerJob{}).Where("run_id = ?", jobId).
		Updates(model.QueryRunnerJob{Status: status, FailureMessage: failureReason})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateQueryRunnerJobNatsSeqNum(
	id uint, seqNum uint64) error {
	tx := db.ORM.
		Model(&model.QueryRunnerJob{}).
		Where("run_id = ?", id).
		Updates(model.QueryRunnerJob{
			NatsSequenceNumber: seqNum,
		})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateTimedOutInProgressQueryRunners() error {
	tx := db.ORM.
		Model(&model.QueryRunnerJob{}).
		Where("status = ?", runner.ComplianceRunnerInProgress).
		Where("updated_at < NOW() - INTERVAL '15 MINUTES'").
		Updates(model.QueryRunnerJob{Status: queryrunner.QueryRunnerTimeOut, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateTimedOutQueuedQueryRunners() error {
	tx := db.ORM.
		Model(&model.QueryRunnerJob{}).
		Where("status = ?", runner.ComplianceRunnerInProgress).
		Where("updated_at < NOW() - INTERVAL '12 HOURS'").
		Updates(model.QueryRunnerJob{Status: queryrunner.QueryRunnerTimeOut, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
