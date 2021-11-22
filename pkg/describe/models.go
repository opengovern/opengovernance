package describe

import (
	"database/sql"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SourceType string

const (
	SourceCloudAWS   SourceType = "AWS"
	SourceCloudAzure SourceType = "Azure"
)

func IsValidSourceType(t SourceType) bool {
	switch t {
	case SourceCloudAWS, SourceCloudAzure:
		return true
	default:
		return false
	}
}

type Source struct {
	gorm.Model
	ID                 uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	Type               SourceType
	Credentials        []byte
	LastDescribedAt    sql.NullTime
	NextDescribeAt     sql.NullTime
	DescribeSourceJobs []DescribeSourceJob `gorm:"foreignKey:SourceID;constraint:OnDelete:CASCADE;"`
}

type DescribeSourceJobStatus string

const (
	DescribeSourceJobCreated              DescribeSourceJobStatus = "CREATED"
	DescribeSourceJobInProgress           DescribeSourceJobStatus = "IN_PROGRESS"
	DescribeSourceJobCompletedWithFailure DescribeSourceJobStatus = "COMPLETED_WITH_FAILURE"
	DescribeSourceJobCompleted            DescribeSourceJobStatus = "COMPLETED"
)

type DescribeSourceJob struct {
	gorm.Model
	SourceID             uuid.UUID             // Not the primary key but should be a unique identifier
	DescribeResourceJobs []DescribeResourceJob `gorm:"foreignKey:ParentJobID;constraint:OnDelete:CASCADE;"`
	Status               DescribeSourceJobStatus
}

type DescribeResourceJobStatus string

const (
	DescribeResourceJobCreated   DescribeResourceJobStatus = "CREATED"
	DescribeResourceJobQueued    DescribeResourceJobStatus = "QUEUED"
	DescribeResourceJobFailed    DescribeResourceJobStatus = "FAILED"
	DescribeResourceJobSucceeded DescribeResourceJobStatus = "SUCCEEDED"
)

type DescribeResourceJob struct {
	gorm.Model
	ParentJobID    uint
	ResourceType   string
	Status         DescribeResourceJobStatus
	FailureMessage string // Should be NULLSTRING
}
