package reporter

import (
	"github.com/jackc/pgtype"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type JobStatus string

const (
	JobStatusPending    JobStatus = "PENDING"
	JobStatusFailed     JobStatus = "FAILED"
	JobStatusSuccessful JobStatus = "SUCCESSFUL"
)

type DatabaseWorkerJob struct {
	gorm.Model
	ConnectionID string `gorm:"index"`
	Status       JobStatus
	JobResults   []WorkerJobResult `gorm:"foreignKey:JobID,references:ID"`
}

type WorkerJobResult struct {
	gorm.Model
	JobID              int
	Query              pgtype.JSONB
	TotalRows          int
	NotMatchingColumns pq.StringArray `gorm:"type:text[];"`
	Mismatches         pgtype.JSONBArray
}
