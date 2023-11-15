package db

import (
	"context"
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	kaytuTrace "github.com/kaytu-io/kaytu-util/pkg/trace"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"
)

func (db Database) CountQueuedDescribeConnectionJobs() (int64, error) {
	var count int64
	tx := db.ORM.Model(&model.DescribeConnectionJob{}).Where("status = ? AND created_at > now() - interval '1 day'", api.DescribeResourceJobQueued).Count(&count)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, tx.Error
	}
	return count, nil
}

func (db Database) CountDescribeConnectionJobsRunOverLast10Minutes() (int64, error) {
	var count int64
	tx := db.ORM.Model(&model.DescribeConnectionJob{}).Where("status != ? AND updated_at > now() - interval '10 minutes'", api.DescribeResourceJobCreated).Count(&count)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, tx.Error
	}
	return count, nil
}

type ResourceTypeCount struct {
	ResourceType string
	Count        int
}

func (db Database) CountRunningDescribeJobsPerResourceType() ([]ResourceTypeCount, error) {
	var count []ResourceTypeCount
	runningJobs := []api.DescribeResourceJobStatus{api.DescribeResourceJobQueued, api.DescribeResourceJobInProgress}
	tx := db.ORM.Raw(`select resource_type, count(*) as count from describe_connection_jobs where status in ? group by 1`, runningJobs).Find(&count)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return count, nil
}

func (db Database) GetLastDescribeConnectionJob(connectionID, resourceType string) (*model.DescribeConnectionJob, error) {
	var job model.DescribeConnectionJob
	tx := db.ORM.Preload(clause.Associations).Where("connection_id = ? AND resource_type = ?", connectionID, resourceType).Order("updated_at DESC").First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}

	return &job, nil
}

func (db Database) GetDescribeConnectionJobByConnectionID(connectionID string) ([]model.DescribeConnectionJob, error) {
	var jobs []model.DescribeConnectionJob
	tx := db.ORM.Preload(clause.Associations).Where("connection_id = ?", connectionID).Find(&jobs)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}

	return jobs, nil
}

func (db Database) GetDescribeConnectionJobByID(id uint) (*model.DescribeConnectionJob, error) {
	var job model.DescribeConnectionJob
	tx := db.ORM.Preload(clause.Associations).Where("id = ?", id).First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}

	return &job, nil
}

func (db Database) RetryDescribeConnectionJob(id uint) error {
	tx := db.ORM.Exec("update describe_connection_jobs set status = ? where id = ?", api.DescribeResourceJobCreated, id)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
func (db Database) QueueDescribeConnectionJob(id uint) error {
	tx := db.ORM.Exec("update describe_connection_jobs set status = ?, queued_at = NOW(), retry_count = retry_count + 1 where id = ?", api.DescribeResourceJobQueued, id)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) ListRandomCreatedDescribeConnectionJobs(ctx context.Context, limit int) ([]model.DescribeConnectionJob, error) {
	ctx, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx, kaytuTrace.GetCurrentFuncName())
	defer span.End()

	var job []model.DescribeConnectionJob

	//runningJobs := []api.D.RawescribeResourceJobStatus{api.DescribeResourceJobQueued, api.DescribeResourceJobInProgress}
	tx := db.ORM.Raw(`
SELECT
	*, random() as r
FROM
	describe_connection_jobs dr
WHERE
	status = ?
ORDER BY r DESC
LIMIT ?
`, api.DescribeResourceJobCreated, limit).Find(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return job, nil
}

func (db Database) GetFailedDescribeConnectionJobs(ctx context.Context) ([]model.DescribeConnectionJob, error) {
	ctx, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx, kaytuTrace.GetCurrentFuncName())
	defer span.End()

	var job []model.DescribeConnectionJob

	tx := db.ORM.Raw(`
SELECT
	*
FROM
	describe_connection_jobs dr
WHERE
	(status = ? OR status = ?) AND
	created_at > now() - interval '2 day' AND
    updated_at < now() - interval '5 minutes' AND
	NOT(error_code IN ('InvalidApiVersionParameter', 'AuthorizationFailed', 'AccessDeniedException', 'InvalidAuthenticationToken', 'AccessDenied', 'InsufficientPrivilegesException', '403', '404', '401', '400')) AND
	(retry_count < 5 OR retry_count IS NULL)
	ORDER BY id DESC
`, api.DescribeResourceJobFailed, api.DescribeResourceJobTimeout).Find(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return job, nil
}

func (db Database) CreateDescribeConnectionJob(job *model.DescribeConnectionJob) error {
	tx := db.ORM.
		Model(&model.DescribeConnectionJob{}).
		Create(job)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) CleanupDescribeConnectionJobsOlderThan(t time.Time) error {
	tx := db.ORM.Where("created_at < ?", t).Unscoped().Delete(&model.DescribeConnectionJob{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

// UpdateDescribeConnectionJobsTimedOut updates the status of DescribeResourceJobs
// that have timed out while in the status of 'CREATED' or 'QUEUED' for longer
// than 4 hours.
func (db Database) UpdateDescribeConnectionJobsTimedOut(describeIntervalHours int64) error {
	tx := db.ORM.
		Model(&model.DescribeConnectionJob{}).
		Where("updated_at < NOW() - INTERVAL '20 minutes'").
		Where("status IN ?", []string{string(api.DescribeResourceJobInProgress)}).
		Updates(model.DescribeConnectionJob{Status: api.DescribeResourceJobTimeout, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	tx = db.ORM.
		Model(&model.DescribeConnectionJob{}).
		Where(fmt.Sprintf("updated_at < NOW() - INTERVAL '%d hours'", describeIntervalHours)).
		Where("status IN ?", []string{string(api.DescribeResourceJobQueued)}).
		Updates(model.DescribeConnectionJob{Status: api.DescribeResourceJobFailed, FailureMessage: "Queued job didn't run"})
	if tx.Error != nil {
		return tx.Error
	}

	tx = db.ORM.
		Model(&model.DescribeConnectionJob{}).
		Where(fmt.Sprintf("updated_at < NOW() - INTERVAL '%d hours'", describeIntervalHours)).
		Where("status IN ?", []string{string(api.DescribeResourceJobCreated)}).
		Updates(model.DescribeConnectionJob{Status: api.DescribeResourceJobFailed, FailureMessage: "Job is aborted"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateResourceTypeDescribeConnectionJobsTimedOut updates the status of DescribeResourceJobs
// that have timed out while in the status of 'CREATED' or 'QUEUED' for longer
// than time interval for the specific resource type.
func (db Database) UpdateResourceTypeDescribeConnectionJobsTimedOut(resourceType string, describeIntervalHours int64) (int, error) {
	totalCount := 0
	tx := db.ORM.
		Model(&model.DescribeConnectionJob{}).
		Where("updated_at < NOW() - INTERVAL '20 minutes'").
		Where("status IN ?", []string{string(api.DescribeResourceJobInProgress)}).
		Where("resource_type = ?", resourceType).
		Updates(model.DescribeConnectionJob{Status: api.DescribeResourceJobTimeout, FailureMessage: "Job timed out", ErrorCode: "JobTimeOut"})
	if tx.Error != nil {
		return totalCount, tx.Error
	}
	tx = db.ORM.
		Model(&model.DescribeConnectionJob{}).
		Where(fmt.Sprintf("updated_at < NOW() - INTERVAL '%d hours'", describeIntervalHours)).
		Where("status IN ?", []string{string(api.DescribeResourceJobQueued)}).
		Where("resource_type = ?", resourceType).
		Updates(model.DescribeConnectionJob{Status: api.DescribeResourceJobFailed, FailureMessage: "Queued job didn't run", ErrorCode: "JobTimeOut"})
	if tx.Error != nil {
		return totalCount, tx.Error
	}
	totalCount += int(tx.RowsAffected)
	tx = db.ORM.
		Model(&model.DescribeConnectionJob{}).
		Where(fmt.Sprintf("updated_at < NOW() - INTERVAL '%d hours'", describeIntervalHours)).
		Where("status IN ?", []string{string(api.DescribeResourceJobCreated)}).
		Where("resource_type = ?", resourceType).
		Updates(model.DescribeConnectionJob{Status: api.DescribeResourceJobFailed, FailureMessage: "Job is aborted", ErrorCode: "JobTimeOut"})
	if tx.Error != nil {
		return totalCount, tx.Error
	}
	totalCount += int(tx.RowsAffected)
	return totalCount, nil
}

// UpdateDescribeConnectionJobStatus updates the status of the DescribeResourceJob to the provided status.
// If the status if 'FAILED', msg could be used to indicate the failure reason
func (db Database) UpdateDescribeConnectionJobStatus(id uint, status api.DescribeResourceJobStatus, msg, errCode string, resourceCount int64) error {
	tx := db.ORM.Exec("UPDATE describe_connection_jobs SET status = ?, failure_message = ?, error_code = ?,  described_resource_count = ? WHERE id = ?",
		status, msg, errCode, resourceCount, id)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateDescribeConnectionJobToInProgress(id uint) error {
	tx := db.ORM.
		Model(&model.DescribeConnectionJob{}).
		Where("id = ?", id).
		Where("status IN ?", []string{string(api.DescribeResourceJobCreated), string(api.DescribeResourceJobQueued)}).
		Updates(model.DescribeConnectionJob{Status: api.DescribeResourceJobInProgress, InProgressedAt: time.Now()})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) GetDescribeStatus(resourceType string) ([]api.DescribeStatus, error) {
	var job []api.DescribeStatus

	tx := db.ORM.Raw(`with conns as (
    select 
        connection_id, max(updated_at) as updated_at 
    from describe_connection_jobs 
    where lower(resource_type) = ? and status in ('SUCCEEDED', 'FAILED', 'TIMEOUT') group by 1
)
select j.connection_id, j.connector, j.status from describe_connection_jobs j inner join conns c on j.connection_id = c.connection_id where j.updated_at = c.updated_at;`, strings.ToLower(resourceType)).Find(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return job, nil
}

func (db Database) GetConnectionDescribeStatus(connectionID string) ([]api.ConnectionDescribeStatus, error) {
	var job []api.ConnectionDescribeStatus

	tx := db.ORM.Raw(`with resourceTypes as (
    select 
        resource_type, max(updated_at) as updated_at 
    from
		describe_connection_jobs 
    where 
		connection_id = ?
	group by 1
)
select 
	j.resource_type, j.status 
from 
	describe_connection_jobs j inner join resourceTypes c on j.resource_type = c.resource_type 
where 
	connection_id = ? AND j.updated_at = c.updated_at;`,
		connectionID, connectionID).Find(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return job, nil
}

func (db Database) CountJobsWithStatus(interval int, connector source.Type, status api.DescribeResourceJobStatus) (*int64, error) {
	var count int64
	query := fmt.Sprintf("SELECT count(*) FROM describe_connection_jobs WHERE (connector = '%s' and created_at > now() - interval '%d hour' and status = '%s') AND deleted_at IS NULL", connector, interval, status)
	tx := db.ORM.Raw(query).Find(&count)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &count, nil
}

func (db Database) ListAllPendingConnection() ([]string, error) {
	var connectionIDs []string

	tx := db.ORM.Raw(`select distinct(connection_id) from describe_connection_jobs where status in ('CREATED', 'QUEDED', 'IN_PROGRESS')`).Find(&connectionIDs)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return connectionIDs, nil
}

func (db Database) ListAllFirstTryPendingConnection() ([]string, error) {
	var connectionIDs []string

	tx := db.ORM.Raw(`select distinct(connection_id) from describe_connection_jobs where status in ('CREATED', 'QUEDED', 'IN_PROGRESS') and retry_count = 0`).Find(&connectionIDs)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return connectionIDs, nil
}

func (db Database) GetLastSuccessfulDescribeJob() (*model.DescribeConnectionJob, error) {
	var job model.DescribeConnectionJob

	tx := db.ORM.Model(&model.DescribeConnectionJob{}).
		Where("status = 'SUCCEEDED'").
		Order("updated_at DESC").First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &job, nil
}

func (db Database) CountJobsAndResources() (int64, int64, error) {
	var count, sum int64
	err := db.ORM.Raw("select count(*), sum(described_resource_count) from describe_connection_jobs").Row().Scan(&count, &sum)
	if err != nil {
		return 0, 0, err
	}
	return count, sum, nil
}
