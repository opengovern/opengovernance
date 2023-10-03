package describe

import (
	"context"
	"errors"
	"fmt"
	kaytuTrace "github.com/kaytu-io/kaytu-util/pkg/trace"
	"go.opentelemetry.io/otel"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/kaytu-io/kaytu-engine/pkg/analytics"
	checkupapi "github.com/kaytu-io/kaytu-engine/pkg/checkup/api"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	insightapi "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer"
	summarizerapi "github.com/kaytu-io/kaytu-engine/pkg/summarizer/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Database struct {
	orm *gorm.DB
}

func (db Database) Initialize() error {
	return db.orm.AutoMigrate(&ComplianceReportJob{}, &InsightJob{}, &CheckupJob{}, &SummarizerJob{},
		&AnalyticsJob{}, &Stack{}, &StackTag{}, &StackEvaluation{},
		&StackCredential{}, &DescribeConnectionJob{},
		&JobSequencer{},
	)
}

// =============================== DescribeConnectionJob ===============================

func (db Database) CreateDescribeConnectionJob(job *DescribeConnectionJob) error {
	tx := db.orm.
		Model(&DescribeConnectionJob{}).
		Create(job)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) CountQueuedDescribeConnectionJobs() (int64, error) {
	var count int64
	tx := db.orm.Model(&DescribeConnectionJob{}).Where("status = ? AND created_at > now() - interval '1 day'", api.DescribeResourceJobQueued).Count(&count)
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
	tx := db.orm.Model(&DescribeConnectionJob{}).Where("status != ? AND updated_at > now() - interval '10 minutes'", api.DescribeResourceJobCreated).Count(&count)
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
	tx := db.orm.Raw(`select resource_type, count(*) as count from describe_connection_jobs where status in ? group by 1`, runningJobs).Find(&count)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return count, nil
}

func (db Database) GetLastDescribeConnectionJob(connectionID, resourceType string) (*DescribeConnectionJob, error) {
	var job DescribeConnectionJob
	tx := db.orm.Preload(clause.Associations).Where("connection_id = ? AND resource_type = ?", connectionID, resourceType).Order("updated_at DESC").First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}

	return &job, nil
}

func (db Database) GetDescribeConnectionJobByConnectionID(connectionID string) ([]DescribeConnectionJob, error) {
	var jobs []DescribeConnectionJob
	tx := db.orm.Preload(clause.Associations).Where("connection_id = ?", connectionID).Find(&jobs)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}

	return jobs, nil
}

func (db Database) QueueDescribeConnectionJob(id uint) error {
	tx := db.orm.Exec("update describe_connection_jobs set status = ?, queued_at = NOW(), retry_count = retry_count + 1 where id = ?", api.DescribeResourceJobQueued, id)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) ListRandomCreatedDescribeConnectionJobs(ctx context.Context, limit int) ([]DescribeConnectionJob, error) {
	ctx, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx, kaytuTrace.GetCurrentFuncName())
	defer span.End()

	var job []DescribeConnectionJob

	//runningJobs := []api.DescribeResourceJobStatus{api.DescribeResourceJobQueued, api.DescribeResourceJobInProgress}
	tx := db.orm.Raw(`
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

func (db Database) GetFailedDescribeConnectionJobs(ctx context.Context, limit int) ([]DescribeConnectionJob, error) {
	ctx, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx, kaytuTrace.GetCurrentFuncName())
	defer span.End()

	var job []DescribeConnectionJob

	tx := db.orm.Raw(`
SELECT
	*
FROM
	describe_connection_jobs dr
WHERE
	status = ? AND
	created_at > now() - interval '3 day' AND 
	NOT(error_code IN ('AccessDeniedException', 'InvalidAuthenticationToken', 'AccessDenied', 'InsufficientPrivilegesException', '403', '404', '401', '400')) AND
	(retry_count < 5 OR retry_count IS NULL)
	ORDER BY created_at DESC LIMIT ?
`, api.DescribeResourceJobFailed, limit).Find(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return job, nil
}

func (db Database) CleanupDescribeConnectionJobsOlderThan(t time.Time) error {
	tx := db.orm.Where("created_at < ?", t).Unscoped().Delete(&DescribeConnectionJob{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

// UpdateDescribeConnectionJobsTimedOut updates the status of DescribeResourceJobs
// that have timed out while in the status of 'CREATED' or 'QUEUED' for longer
// than 4 hours.
func (db Database) UpdateDescribeConnectionJobsTimedOut(describeIntervalHours int64) error {
	tx := db.orm.
		Model(&DescribeConnectionJob{}).
		Where("updated_at < NOW() - INTERVAL '20 minutes'").
		Where("status IN ?", []string{string(api.DescribeResourceJobInProgress)}).
		Updates(DescribeConnectionJob{Status: api.DescribeResourceJobTimeout, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	tx = db.orm.
		Model(&DescribeConnectionJob{}).
		Where(fmt.Sprintf("updated_at < NOW() - INTERVAL '%d hours'", describeIntervalHours)).
		Where("status IN ?", []string{string(api.DescribeResourceJobQueued)}).
		Updates(DescribeConnectionJob{Status: api.DescribeResourceJobFailed, FailureMessage: "Queued job didn't run"})
	if tx.Error != nil {
		return tx.Error
	}

	tx = db.orm.
		Model(&DescribeConnectionJob{}).
		Where(fmt.Sprintf("updated_at < NOW() - INTERVAL '%d hours'", describeIntervalHours)).
		Where("status IN ?", []string{string(api.DescribeResourceJobCreated)}).
		Updates(DescribeConnectionJob{Status: api.DescribeResourceJobFailed, FailureMessage: "Job is aborted"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateResourceTypeDescribeConnectionJobsTimedOut updates the status of DescribeResourceJobs
// that have timed out while in the status of 'CREATED' or 'QUEUED' for longer
// than time interval for the specific resource type.
func (db Database) UpdateResourceTypeDescribeConnectionJobsTimedOut(resourceType string, describeIntervalHours int64) error {
	tx := db.orm.
		Model(&DescribeConnectionJob{}).
		Where("updated_at < NOW() - INTERVAL '20 minutes'").
		Where("status IN ?", []string{string(api.DescribeResourceJobInProgress)}).
		Where("resource_type = ?", resourceType).
		Updates(DescribeConnectionJob{Status: api.DescribeResourceJobTimeout, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	tx = db.orm.
		Model(&DescribeConnectionJob{}).
		Where(fmt.Sprintf("updated_at < NOW() - INTERVAL '%d hours'", describeIntervalHours)).
		Where("status IN ?", []string{string(api.DescribeResourceJobQueued)}).
		Where("resource_type = ?", resourceType).
		Updates(DescribeConnectionJob{Status: api.DescribeResourceJobFailed, FailureMessage: "Queued job didn't run"})
	if tx.Error != nil {
		return tx.Error
	}

	tx = db.orm.
		Model(&DescribeConnectionJob{}).
		Where(fmt.Sprintf("updated_at < NOW() - INTERVAL '%d hours'", describeIntervalHours)).
		Where("status IN ?", []string{string(api.DescribeResourceJobCreated)}).
		Where("resource_type = ?", resourceType).
		Updates(DescribeConnectionJob{Status: api.DescribeResourceJobFailed, FailureMessage: "Job is aborted"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateDescribeConnectionJobStatus updates the status of the DescribeResourceJob to the provided status.
// If the status if 'FAILED', msg could be used to indicate the failure reason
func (db Database) UpdateDescribeConnectionJobStatus(id uint, status api.DescribeResourceJobStatus, msg, errCode string, resourceCount int64) error {
	tx := db.orm.
		Model(&DescribeConnectionJob{}).
		Where("id = ?", id).
		Updates(DescribeConnectionJob{Status: status, FailureMessage: msg, ErrorCode: errCode, DescribedResourceCount: resourceCount})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateDescribeConnectionJobToInProgress(id uint) error {
	tx := db.orm.
		Model(&DescribeConnectionJob{}).
		Where("id = ?", id).
		Where("status IN ?", []string{string(api.DescribeResourceJobCreated), string(api.DescribeResourceJobQueued)}).
		Updates(DescribeConnectionJob{Status: api.DescribeResourceJobInProgress, InProgressedAt: time.Now()})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) GetDescribeStatus(resourceType string) ([]api.DescribeStatus, error) {
	var job []api.DescribeStatus

	tx := db.orm.Raw(`with conns as (
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

	tx := db.orm.Raw(`with resourceTypes as (
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

func (db Database) ListAllPendingConnection() ([]string, error) {
	var connectionIDs []string

	tx := db.orm.Raw(`select distinct(connection_id) from describe_connection_jobs where status in ('CREATED', 'QUEDED', 'IN_PROGRESS')`).Find(&connectionIDs)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return connectionIDs, nil
}

//
//func (db Database) ListPendingDescribeResourceJobs() ([]DescribeResourceJob, error) {
//	var jobs []DescribeResourceJob
//	tx := db.orm.Where("status in (?, ?)", api.DescribeResourceJobQueued, api.DescribeResourceJobCreated).Find(&jobs)
//	if tx.Error != nil {
//		return nil, tx.Error
//	}
//
//	return jobs, nil
//}
//
//func (db Database) FetchRandomCreatedDescribeResourceJobs(parentIdExceptionList []uint) (*DescribeResourceJob, error) {
//	var job DescribeResourceJob
//	tx := db.orm.Where("status = ?", api.DescribeResourceJobCreated)
//
//	if len(parentIdExceptionList) > 0 {
//		tx = tx.Where("NOT(parent_job_id IN ?)", parentIdExceptionList)
//	}
//
//	tx = tx.Order("random()").First(&job)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return nil, nil
//		}
//		return nil, tx.Error
//	}
//	return &job, nil
//}
//
//
//
//func (db Database) CountQueuedDescribeResourceJobs(ctx context.Context) (int64, error) {
//	ctx, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx, kaytuTrace.GetCurrentFuncName())
//	defer span.End()
//
//	var count int64
//	tx := db.orm.Model(&DescribeConnectionJob{}).Where("status = ? AND created_at > now() - interval '1 day'", api.DescribeResourceJobQueued).Count(&count)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return 0, nil
//		}
//		return 0, tx.Error
//	}
//	return count, nil
//}
//
//func (db Database) GetLastDescribeConnectionJob(connectionID, resourceType string) (*DescribeConnectionJob, error) {
//	var job DescribeConnectionJob
//	tx := db.orm.Preload(clause.Associations).Where("connection_id = ? AND resource_type = ?", connectionID, resourceType).Order("updated_at DESC").First(&job)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return nil, nil
//		}
//		return nil, tx.Error
//	}
//
//	return &job, nil
//}

//// UpdateDescribeSourceJob updates the DescribeSourceJob status.
//func (db Database) UpdateDescribeSourceJob(id uint, status api.DescribeSourceJobStatus) error {
//	tx := db.orm.
//		Model(&DescribeSourceJob{}).
//		Where("id = ?", id).
//		Updates(DescribeSourceJob{Status: status})
//	if tx.Error != nil {
//		return tx.Error
//	}
//
//	return nil
//}
//
//// ListAllDescribeSourceJobs lists all DescribeSourceJob .
//func (db Database) ListAllDescribeSourceJobs() ([]DescribeSourceJob, error) {
//	var jobs []DescribeSourceJob
//	tx := db.orm.Find(&jobs)
//	if tx.Error != nil {
//		return nil, tx.Error
//	}
//
//	return jobs, nil
//}
//
//func (db Database) ListPendingDescribeSourceJobs() ([]DescribeSourceJob, error) {
//	var jobs []DescribeSourceJob
//	tx := db.orm.Where("status in (?, ?)", api.DescribeSourceJobInProgress, api.DescribeSourceJobCreated).Find(&jobs)
//	if tx.Error != nil {
//		return nil, tx.Error
//	}
//
//	return jobs, nil
//}
//
//func (db Database) CleanupDescribeSourceJobsOlderThan(t time.Time) error {
//	tx := db.orm.Where("created_at < ?", t).Unscoped().Delete(&DescribeSourceJob{})
//	if tx.Error != nil {
//		return tx.Error
//	}
//	return nil
//}
//
//func (db Database) ListPendingDescribeResourceJobs() ([]DescribeResourceJob, error) {
//	var jobs []DescribeResourceJob
//	tx := db.orm.Where("status in (?, ?)", api.DescribeResourceJobQueued, api.DescribeResourceJobCreated).Find(&jobs)
//	if tx.Error != nil {
//		return nil, tx.Error
//	}
//
//	return jobs, nil
//}
//
//func (db Database) FetchRandomCreatedDescribeResourceJobs(parentIdExceptionList []uint) (*DescribeResourceJob, error) {
//	var job DescribeResourceJob
//	tx := db.orm.Where("status = ?", api.DescribeResourceJobCreated)
//
//	if len(parentIdExceptionList) > 0 {
//		tx = tx.Where("NOT(parent_job_id IN ?)", parentIdExceptionList)
//	}
//
//	tx = tx.Order("random()").First(&job)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return nil, nil
//		}
//		return nil, tx.Error
//	}
//	return &job, nil
//}
//
//func (db Database) ListRandomCreatedDescribeResourceJobs(limit int) ([]DescribeResourceJob, error) {
//	var job []DescribeResourceJob
//
//	runningJobs := []api.DescribeResourceJobStatus{api.DescribeResourceJobQueued, api.DescribeResourceJobInProgress}
//	tx := db.orm.Raw(`
//SELECT
// *
//FROM (
//  SELECT
//   ROW_NUMBER() OVER (PARTITION BY resource_type) AS r,
//   t.*
//  FROM (
//      SELECT
//       *
//      FROM
//        describe_resource_jobs dr
//      WHERE
//        status = ? AND
//        (select count(*) from describe_resource_jobs where parent_job_id = dr.parent_job_id AND status IN ?) <= 10
//  ) AS t) AS rowed
//WHERE
//  rowed.r <= 5
//LIMIT ?
//`, api.DescribeResourceJobCreated, runningJobs, limit).Find(&job)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return nil, nil
//		}
//		return nil, tx.Error
//	}
//	return job, nil
//}
//
//func (db Database) GetFailedDescribeResourceJobs() ([]DescribeResourceJob, error) {
//	var job []DescribeResourceJob
//
//	tx := db.orm.Raw(`
//SELECT
//	*
//FROM
//	describe_resource_jobs dr
//WHERE
//	status = ? AND
//	NOT(error_code IN ('AccessDeniedException', 'InvalidAuthenticationToken', 'AccessDenied', 'InsufficientPrivilegesException', '403', '404', '401', '400')) AND
//	(retry_count < 3 OR retry_count IS NULL) AND
//	(select count(*) from describe_resource_jobs where parent_job_id = dr.parent_job_id AND status IN (?, ?)) = 0
//	LIMIT 10
//`, api.DescribeResourceJobFailed, api.DescribeResourceJobQueued, api.DescribeResourceJobInProgress).Find(&job)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return nil, nil
//		}
//		return nil, tx.Error
//	}
//	return job, nil
//}
//
//func (db Database) CountQueuedDescribeResourceJobs() (int64, error) {
//	var count int64
//	tx := db.orm.Model(&DescribeResourceJob{}).Where("status = ? AND created_at > now() - interval '1 day'", api.DescribeResourceJobQueued).Count(&count)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return 0, nil
//		}
//		return 0, tx.Error
//	}
//	return count, nil
//}
//
//func (db Database) CountQueuedInProgressDescribeResourceJobsByParentID(id uint) (int64, error) {
//	var count int64
//	tx := db.orm.Model(&DescribeResourceJob{}).Where("status IN (?, ?) AND parent_job_id = ? AND created_at > now() - interval '1 day'", api.DescribeResourceJobQueued, api.DescribeResourceJobInProgress, id).Count(&count)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return 0, nil
//		}
//		return 0, tx.Error
//	}
//	return count, nil
//}
//
//
//func (db Database) ListCreatedDescribeSourceJobs() ([]DescribeSourceJob, error) {
//	var jobs []DescribeSourceJob
//	tx := db.orm.Where("status in (?)", api.DescribeSourceJobCreated).Find(&jobs)
//	if tx.Error != nil {
//		return nil, tx.Error
//	}
//
//	return jobs, nil
//}

func (db Database) ListPendingSummarizeJobs() ([]SummarizerJob, error) {
	var jobs []SummarizerJob
	tx := db.orm.Where("status = ?", summarizerapi.SummarizerJobInProgress).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return jobs, nil
}

func (db Database) ListPendingInsightJobs() ([]InsightJob, error) {
	var jobs []InsightJob
	tx := db.orm.Where("status = ?", insightapi.InsightJobInProgress).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return jobs, nil
}

func (db Database) CleanupInsightJobsOlderThan(t time.Time) error {
	tx := db.orm.Where("created_at < ?", t).Unscoped().Delete(&InsightJob{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

//
//// ListDescribeSourceJobs lists the DescribeSourceJobs for the given sourcel.
//func (db Database) ListDescribeSourceJobs(sourceID uuid.UUID) ([]DescribeSourceJob, error) {
//	var jobs []DescribeSourceJob
//	tx := db.orm.Preload(clause.Associations).Where("source_id = ?", sourceID).Find(&jobs)
//	if tx.Error != nil {
//		return nil, tx.Error
//	}
//
//	return jobs, nil
//}
//
//// GetLastDescribeSourceJob returns the last DescribeSourceJobs for the given source.
//func (db Database) GetLastDescribeSourceJob(sourceID string) (*DescribeSourceJob, error) {
//	var job DescribeSourceJob
//	tx := db.orm.Preload(clause.Associations).Where("source_id = ?", sourceID).Order("updated_at DESC").First(&job)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return nil, nil
//		}
//		return nil, tx.Error
//	}
//
//	return &job, nil
//}
//
//func (db Database) GetLastFullDiscoveryDescribeSourceJob(sourceID string) (*DescribeSourceJob, error) {
//	var job DescribeSourceJob
//	tx := db.orm.Preload(clause.Associations).Where("source_id = ? AND full_discovery = true", sourceID).Order("updated_at DESC").First(&job)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return nil, nil
//		}
//		return nil, tx.Error
//	}
//
//	return &job, nil
//}
//
//// GetDescribeSourceJob returns the DescribeSourceJobs for the given id.
//func (db Database) GetDescribeSourceJob(ctx context.Context, jobID uint) (*DescribeSourceJob, error) {
//	ctx, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx, kaytuTrace.GetCurrentFuncName())
//	defer span.End()
//
//	var job DescribeSourceJob
//	tx := db.orm.Preload(clause.Associations).Where("id = ?", jobID).First(&job)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return nil, nil
//		}
//		return nil, tx.Error
//	}
//
//	return &job, nil
//}
//
//// GetOldCompletedSourceJob returns the last DescribeSourceJobs for the given source at nDaysBefore
//func (db Database) GetOldCompletedSourceJob(sourceID uuid.UUID, nDaysBefore int) (*DescribeSourceJob, error) {
//	var job *DescribeSourceJob
//	tx := db.orm.Model(&DescribeSourceJob{}).
//		Where("status in ?", []string{string(api.DescribeSourceJobCompleted), string(api.DescribeSourceJobCompletedWithFailure)}).
//		Where("source_id = ?", sourceID.String()).
//		Where(fmt.Sprintf("updated_at < now() - interval '%d days'", nDaysBefore-1)).
//		Where(fmt.Sprintf("updated_at >= now() - interval '%d days'", nDaysBefore)).
//		Order("updated_at DESC").
//		First(&job)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return nil, nil
//		}
//		return nil, tx.Error
//	} else if tx.RowsAffected != 1 {
//		return nil, nil
//	}
//	return job, nil
//}
//
//type DescribedSourceJobDescribeResourceJobStatus struct {
//	DescribeSourceJobID       uint                          `gorm:"column:id"`
//	DescribeSourceStatus      api.DescribeSourceJobStatus   `gorm:"column:dsstatus"`
//	DescribeResourceJobStatus api.DescribeResourceJobStatus `gorm:"column:status"`
//	DescribeResourceJobCount  int                           `gorm:"column:count"`
//}

//
//// Finds the DescribeSourceJobs that are IN_PROGRESS and find the
//// status of the corresponding DescribeResourceJobs and their counts.
//func (db Database) QueryInProgressDescribedSourceJobGroupByDescribeResourceJobStatus() ([]DescribedSourceJobDescribeResourceJobStatus, error) {
//	var results []DescribedSourceJobDescribeResourceJobStatus
//
//	tx := db.orm.
//		Model(&DescribeSourceJob{}).
//		Select("describe_source_jobs.id, describe_source_jobs.status as dsstatus, describe_resource_jobs.status, COUNT(*)").
//		Joins("JOIN describe_resource_jobs ON describe_source_jobs.id = describe_resource_jobs.parent_job_id").
//		Where("describe_source_jobs.status IN ?", []string{string(api.DescribeSourceJobCreated), string(api.DescribeSourceJobInProgress)}).
//		Group("describe_source_jobs.id").
//		Group("describe_resource_jobs.status").
//		Order("describe_source_jobs.id ASC").
//		Find(&results)
//	if tx.Error != nil {
//		return nil, tx.Error
//	}
//
//	return results, nil
//}
//
//func (db Database) QueryOlderThanNRecentCompletedDescribeSourceJobs(n int) ([]DescribeSourceJob, error) {
//	var results []DescribeSourceJob
//
//	tx := db.orm.Raw(
//		`
//SELECT jobs.id
//FROM (
//	SELECT *, rank() OVER (
//		PARTITION BY source_id
//		ORDER BY updated_at DESC
//	)
//	FROM describe_source_jobs
//	WHERE status IN ? AND deleted_at IS NULL)
//jobs
//WHERE rank > ?
//`, []string{string(api.DescribeSourceJobCompleted), string(api.DescribeSourceJobCompletedWithFailure)}, n).Scan(&results)
//
//	if tx.Error != nil {
//		return nil, tx.Error
//	}
//
//	return results, nil
//}
//
//func (db Database) QueryDescribeSourceJobs(id string) ([]DescribeSourceJob, error) {
//	status := []string{string(api.DescribeSourceJobCompleted), string(api.DescribeSourceJobCompletedWithFailure)}
//
//	var jobs []DescribeSourceJob
//	tx := db.orm.Where("status IN ? AND deleted_at IS NULL AND source_id = ?", status, id).Find(&jobs)
//	if tx.Error != nil {
//		return nil, tx.Error
//	}
//	return jobs, nil
//}
//
//func (db Database) DeleteDescribeSourceJob(id uint) error {
//	tx := db.orm.
//		Where("id = ?", id).
//		Unscoped().
//		Delete(&DescribeSourceJob{})
//	if tx.Error != nil {
//		return tx.Error
//	} else if tx.RowsAffected != 1 {
//		return fmt.Errorf("delete source: didn't find the describe source job to delete")
//	}
//
//	return nil
//}
//
//// =============================== CloudNativeDescribeSourceJob ===============================
//
//// CreateCloudNativeDescribeSourceJob creates a new CloudNativeDescribeSourceJob.
//// If there is no error, the job is updated with the assigned ID
//func (db Database) CreateCloudNativeDescribeSourceJob(job *CloudNativeDescribeSourceJob) error {
//	tx := db.orm.
//		Model(&CloudNativeDescribeSourceJob{}).
//		Create(job)
//	if tx.Error != nil {
//		return tx.Error
//	}
//
//	return nil
//}
//
//func (db Database) GetCloudNativeDescribeSourceJob(jobID string) (*CloudNativeDescribeSourceJob, error) {
//	var job CloudNativeDescribeSourceJob
//	tx := db.orm.Preload(clause.Associations).Where("job_id = ?", jobID).First(&job)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return nil, nil
//		}
//		return nil, tx.Error
//	}
//
//	return &job, nil
//}
//
//func (db Database) GetCloudNativeDescribeSourceJobBySourceJobID(jobID uint) (*CloudNativeDescribeSourceJob, error) {
//	var job CloudNativeDescribeSourceJob
//	tx := db.orm.Preload(clause.Associations).Where("source_job_id = ?", jobID).First(&job)
//	if tx.Error != nil {
//		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
//			return nil, nil
//		}
//		return nil, tx.Error
//	}
//
//	return &job, nil
//}
//
//func (db Database) CleanupCloudNativeDescribeSourceJobsOlderThan(t time.Time) error {
//	tx := db.orm.Where("created_at < ?", t).Unscoped().Delete(&CloudNativeDescribeSourceJob{})
//	if tx.Error != nil {
//		return tx.Error
//	}
//
//	return nil
//}
//
//// =============================== DescribeResourceJob ===============================
//
//
//func (db Database) QueueDescribeResourceJob(id uint) error {
//	tx := db.orm.Exec("update describe_resource_jobs set status = ?, retry_count = retry_count + 1 where id = ?", api.DescribeResourceJobQueued, id)
//	if tx.Error != nil {
//		return tx.Error
//	}
//
//	return nil
//}
//
//func (db Database) UpdateDescribeResourceJobToInProgress(id uint) error {
//	tx := db.orm.
//		Model(&DescribeResourceJob{}).
//		Where("id = ?", id).
//		Where("status IN ?", []string{string(api.DescribeResourceJobCreated), string(api.DescribeResourceJobQueued)}).
//		Updates(DescribeResourceJob{Status: api.DescribeResourceJobInProgress})
//	if tx.Error != nil {
//		return tx.Error
//	}
//
//	return nil
//}
//

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
func (db Database) UpdateComplianceReportJobsTimedOut(complianceIntervalHours int64) error {
	tx := db.orm.
		Model(&ComplianceReportJob{}).
		Where(fmt.Sprintf("created_at < NOW() - INTERVAL '%d HOURS'", complianceIntervalHours)).
		Where("status IN ?", []string{string(api2.ComplianceReportJobCreated), string(api2.ComplianceReportJobInProgress)}).
		Updates(ComplianceReportJob{Status: api2.ComplianceReportJobCompletedWithFailure, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// ListComplianceReportJobs lists the ComplianceReportJob .
func (db Database) ListComplianceReportJobs(sourceID uuid.UUID) ([]ComplianceReportJob, error) {
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

func (db Database) GetComplianceReportJobByID(ID uint) (*ComplianceReportJob, error) {
	var job ComplianceReportJob
	tx := db.orm.Where("id = ?", ID).Find(&job)
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

func (db Database) ListComplianceReportsWithFilter(timeAfter, timeBefore *time.Time, connectionID *string, connector *source.Type, benchmarkID *string) ([]ComplianceReportJob, error) {
	var jobs []ComplianceReportJob
	tx := db.orm
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
	tx = tx.Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return jobs, nil
}

func (db Database) CleanupComplianceReportJobsOlderThan(t time.Time) error {
	tx := db.orm.Where("updated_at < ?", t).Unscoped().Delete(&ComplianceReportJob{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) DeleteComplianceReportJob(id uint) error {
	tx := db.orm.
		Where("id = ?", id).
		Unscoped().
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
`, []string{string(api2.ComplianceReportJobCompleted), string(api2.ComplianceReportJobCompletedWithFailure)}, n).Scan(&results)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return results, nil
}

func (db Database) QueryComplianceReportJobs(id string) ([]ComplianceReportJob, error) {
	status := []string{string(api2.ComplianceReportJobCompleted), string(api2.ComplianceReportJobCompletedWithFailure)}

	var jobs []ComplianceReportJob
	tx := db.orm.Where("status IN ? AND deleted_at IS NULL AND source_id = ?", status, id).Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return jobs, nil
}

func (db Database) AddInsightJob(job *InsightJob) error {
	tx := db.orm.Model(&InsightJob{}).
		Create(job)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateInsightJobStatus(job InsightJob) error {
	tx := db.orm.Model(&InsightJob{}).
		Where("id = ?", job.ID).
		Update("status", job.Status)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateInsightJob(jobID uint, status insightapi.InsightJobStatus, failedMessage string) error {
	tx := db.orm.Model(&InsightJob{}).
		Where("id = ?", jobID).
		Updates(InsightJob{
			Status:         status,
			FailureMessage: failedMessage,
		})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) FetchLastInsightJob() (*InsightJob, error) {
	var job InsightJob
	tx := db.orm.Model(&InsightJob{}).
		Order("created_at DESC").First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &job, nil
}

func (db Database) GetLastInsightJob(insightID uint, sourceID string) (*InsightJob, error) {
	var job InsightJob
	tx := db.orm.Model(&InsightJob{}).
		Where("source_id = ? AND insight_id = ?", sourceID, insightID).
		Order("created_at DESC").First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &job, nil
}

func (db Database) ListInsightJobs() ([]InsightJob, error) {
	var job []InsightJob
	tx := db.orm.Model(&InsightJob{}).Find(&job)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return job, nil
}

func (db Database) GetOldCompletedInsightJob(insightID uint, nDaysBefore int) (*InsightJob, error) {
	var job *InsightJob
	tx := db.orm.Model(&InsightJob{}).
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

func (db Database) GetInsightJobById(jobId uint) (*InsightJob, error) {
	var job InsightJob
	tx := db.orm.Model(&InsightJob{}).
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

// UpdateInsightJobsTimedOut updates the status of InsightJobs
// that have timed out while in the status of 'IN_PROGRESS' for longer
// than 4 hours.
func (db Database) UpdateInsightJobsTimedOut(insightIntervalHours int64) error {
	tx := db.orm.
		Model(&InsightJob{}).
		Where(fmt.Sprintf("created_at < NOW() - INTERVAL '%d HOURS'", insightIntervalHours*2)).
		Where("status IN ?", []string{string(insightapi.InsightJobInProgress)}).
		Updates(InsightJob{Status: insightapi.InsightJobFailed, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) AddCheckupJob(job *CheckupJob) error {
	tx := db.orm.Model(&CheckupJob{}).
		Create(job)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateCheckupJobStatus(job CheckupJob) error {
	tx := db.orm.Model(&CheckupJob{}).
		Where("id = ?", job.ID).
		Update("status", job.Status)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateCheckupJob(jobID uint, status checkupapi.CheckupJobStatus, failedMessage string) error {
	tx := db.orm.Model(&CheckupJob{}).
		Where("id = ?", jobID).
		Updates(CheckupJob{
			Status:         status,
			FailureMessage: failedMessage,
		})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) FetchLastCheckupJob() (*CheckupJob, error) {
	var job CheckupJob
	tx := db.orm.Model(&CheckupJob{}).
		Order("created_at DESC").First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &job, nil
}

func (db Database) ListCheckupJobs() ([]CheckupJob, error) {
	var job []CheckupJob
	tx := db.orm.Model(&CheckupJob{}).Find(&job)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return job, nil
}

// UpdateCheckupJobsTimedOut updates the status of CheckupJobs
// that have timed out while in the status of 'IN_PROGRESS' for longer
// than checkupIntervalHours hours.
func (db Database) UpdateCheckupJobsTimedOut(checkupIntervalHours int64) error {
	tx := db.orm.
		Model(&CheckupJob{}).
		Where(fmt.Sprintf("created_at < NOW() - INTERVAL '%d HOURS'", checkupIntervalHours*2)).
		Where("status IN ?", []string{string(checkupapi.CheckupJobInProgress)}).
		Updates(CheckupJob{Status: checkupapi.CheckupJobFailed, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateSummarizerJobsTimedOut updates the status of InsightJobs
// that have timed out while in the status of 'IN_PROGRESS' for longer
// than 4 hours.
func (db Database) UpdateSummarizerJobsTimedOut(summarizerIntervalHours int64) error {
	tx := db.orm.
		Model(&SummarizerJob{}).
		Where(fmt.Sprintf("created_at < NOW() - INTERVAL '%d HOURS'", summarizerIntervalHours*2)).
		Where("status IN ?", []string{string(summarizerapi.SummarizerJobInProgress)}).
		Updates(SummarizerJob{Status: summarizerapi.SummarizerJobFailed, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateSummarizerJob(jobID uint, status summarizerapi.SummarizerJobStatus, failedMessage string) error {
	tx := db.orm.Model(&SummarizerJob{}).
		Where("id = ?", jobID).
		Updates(SummarizerJob{
			Status:         status,
			FailureMessage: failedMessage,
		})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) AddSummarizerJob(job *SummarizerJob) error {
	tx := db.orm.Model(&SummarizerJob{}).
		Create(job)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateSummarizerJobStatus(job SummarizerJob) error {
	tx := db.orm.Model(&SummarizerJob{}).
		Where("id = ?", job.ID).
		Update("status", job.Status)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) GetOngoingSummarizerJobsByType(jobType summarizer.JobType) ([]SummarizerJob, error) {
	var jobs []SummarizerJob
	tx := db.orm.Model(&SummarizerJob{}).
		Where("job_type = ?", jobType).
		Where("status = ?", summarizerapi.SummarizerJobInProgress).
		Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return jobs, nil
}

func (db Database) FetchLastSummarizerJob(jobType summarizer.JobType) (*SummarizerJob, error) {
	var job SummarizerJob
	tx := db.orm.Model(&SummarizerJob{}).
		Where("job_type = ?", jobType).
		Order("created_at DESC").First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &job, nil
}

func (db Database) FetchLastAnalyticsJob() (*AnalyticsJob, error) {
	var job AnalyticsJob
	tx := db.orm.Model(&AnalyticsJob{}).
		Order("created_at DESC").First(&job)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &job, nil
}

func (db Database) AddAnalyticsJob(job *AnalyticsJob) error {
	tx := db.orm.Model(&AnalyticsJob{}).
		Create(job)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateAnalyticsJobStatus(job AnalyticsJob) error {
	tx := db.orm.Model(&AnalyticsJob{}).
		Where("id = ?", job.ID).
		Update("status", job.Status)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateAnalyticsJobsTimedOut(analyticsIntervalHours int64) error {
	tx := db.orm.
		Model(&AnalyticsJob{}).
		Where(fmt.Sprintf("created_at < NOW() - INTERVAL '%d HOURS'", analyticsIntervalHours*2)).
		Where("status IN ?", []string{string(analytics.JobInProgress)}).
		Updates(AnalyticsJob{Status: analytics.JobCompletedWithFailure, FailureMessage: "Job timed out"})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateAnalyticsJob(jobID uint, status analytics.JobStatus, failedMessage string) error {
	tx := db.orm.Model(&AnalyticsJob{}).
		Where("id = ?", jobID).
		Updates(AnalyticsJob{
			Status:         status,
			FailureMessage: failedMessage,
		})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

type GetLatestSuccessfulDescribeJobIDsPerResourcePerAccountResult struct {
	ResourceType  string `gorm:"column:resource_type"`
	ResourceJobID uint   `gorm:"column:resource_job_id"`
}

// ===========================================STACK===============================================

func (db Database) AddStack(record *Stack) error {
	return db.orm.Model(&Stack{}).
		Create(record).Error
}

func (db Database) GetStack(stackID string) (Stack, error) {
	var s Stack
	tx := db.orm.Model(&Stack{}).
		Where("stack_id = ?", stackID).
		Preload("Tags").
		Preload("Evaluations").
		First(&s)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return Stack{}, nil
		}
		return Stack{}, tx.Error
	}

	return s, nil
}

func (db Database) ListStacks(tags map[string][]string, accountIds []string) ([]Stack, error) {
	var s []Stack
	query := db.orm.Model(&Stack{}).
		Preload("Tags")
	if len(accountIds) != 0 {
		query = query.Where("EXISTS (SELECT 1 FROM unnest(account_ids) AS account WHERE account IN ?)", accountIds)
	}
	if len(tags) != 0 {
		query = query.Joins("JOIN stack_tags AS tags ON tags.stack_id = stacks.stack_id")
		for key, values := range tags {
			if len(values) != 0 {
				query = query.Where("tags.key = ? AND (tags.value && ?)", key, pq.StringArray(values))
			} else {
				query = query.Where("tags.key = ?", key)
			}
		}
	}

	tx := query.Find(&s)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) DeleteStack(stackID string) error {
	return db.orm.Model(&Stack{}).
		Where("stack_id = ?", stackID).
		Delete(&Stack{}).Error
}

func (db Database) UpdateStackResources(stackID string, resources pq.StringArray) error {
	tx := db.orm.Model(&Stack{}).
		Where("stack_id = ?", stackID).
		Update("resources", resources)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) AddEvaluation(record *StackEvaluation) error {
	return db.orm.Model(&StackEvaluation{}).
		Create(record).Error
}

func (db Database) GetEvaluation(jobId uint) (StackEvaluation, error) {
	var result StackEvaluation
	tx := db.orm.Model(&StackEvaluation{}).
		Where("job_id = ?", jobId).
		First(&result)
	if tx.Error != nil {
		return StackEvaluation{}, tx.Error
	}

	return result, nil
}

func (db Database) GetResourceStacks(resourceID string) ([]Stack, error) {
	var stacks []Stack
	tx := db.orm.Model(&Stack{}).
		Where("resources @> ?", pq.StringArray{resourceID}).
		Preload("Tags").
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}

func (db Database) CreateStackCredential(a *StackCredential) error {
	tx := db.orm.
		Model(&StackCredential{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "stack_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"secret"}),
		}).
		Create(a)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) GetStackCredential(stackId string) (StackCredential, error) {
	var stackCredential StackCredential
	tx := db.orm.Model(&StackCredential{}).
		Where("stack_id = ?", stackId).
		Find(&stackCredential)
	if tx.Error != nil {
		return StackCredential{}, tx.Error
	}
	return stackCredential, nil
}

func (db Database) RemoveStackCredential(stackId string) error {
	tx := db.orm.Model(&StackCredential{}).
		Where("stack_id = ?", stackId).
		Delete(&StackCredential{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) ListPendingStacks() ([]Stack, error) {
	var stacks []Stack
	tx := db.orm.Model(&Stack{}).
		Where("status IN (?, ?)", api.StackStatusPending, api.StackStatusStalled).
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}

func (db Database) ListCreatedStacks() ([]Stack, error) {
	var stacks []Stack
	tx := db.orm.Model(&Stack{}).
		Where("status = ?", api.StackStatusCreated).
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}

func (db Database) UpdateStackStatus(stackId string, status api.StackStatus) error {
	tx := db.orm.Model(&Stack{}).
		Where("stack_id = ?", stackId).
		Update("status", status)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateStackFailureMessage(stackId string, message string) error {
	tx := db.orm.Model(&Stack{}).
		Where("stack_id = ?", stackId).
		Update("failure_message", message)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) ListDescribingStacks() ([]Stack, error) {
	var stacks []Stack
	tx := db.orm.Model(&Stack{}).
		Where("status = ?", api.StackStatusDescribing).
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}

func (db Database) ListDescribedStacks() ([]Stack, error) {
	var stacks []Stack
	tx := db.orm.Model(&Stack{}).
		Where("status = ?", api.StackStatusDescribed).
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}

func (db Database) ListEvaluatingStacks() ([]Stack, error) {
	var stacks []Stack
	tx := db.orm.Model(&Stack{}).
		Where("status = ?", api.StackStatusEvaluating).
		Preload("Evaluations").
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}

func (db Database) UpdateEvaluationStatus(jobId uint, status api.StackEvaluationStatus) error {
	tx := db.orm.Model(&StackEvaluation{}).
		Where("job_id = ?", jobId).
		Update("status", status)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) ListFailedStacks() ([]Stack, error) {
	var stacks []Stack
	tx := db.orm.Model(&Stack{}).
		Where("status = ?", api.StackStatusFailed).
		Preload("Evaluations").
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}

func (db Database) CreateJobSequencer(job *JobSequencer) error {
	tx := db.orm.
		Model(&JobSequencer{}).
		Create(job)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) ListWaitingJobSequencers() ([]JobSequencer, error) {
	var jobs []JobSequencer
	tx := db.orm.Model(&JobSequencer{}).
		Where("status = ?", JobSequencerWaitingForDependencies).
		Find(&jobs)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return jobs, nil

}

func (db Database) UpdateJobSequencerFailed(id uint) error {
	tx := db.orm.Model(&JobSequencer{}).
		Where("id = ?", id).
		Update("status", JobSequencerFailed)
	if tx.Error != nil {
		return tx.Error
	}
	return nil

}

func (db Database) UpdateJobSequencerFinished(id uint) error {
	tx := db.orm.Model(&JobSequencer{}).
		Where("id = ?", id).
		Update("status", JobSequencerFinished)
	if tx.Error != nil {
		return tx.Error
	}
	return nil

}
