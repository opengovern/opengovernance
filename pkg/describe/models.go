package describe

import (
	"database/sql"
	"time"

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/api"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gorm.io/gorm"
)

type Source struct {
	gorm.Model
	ID                     uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	Type                   api.SourceType
	ConfigRef              string
	LastDescribedAt        sql.NullTime
	NextDescribeAt         sql.NullTime
	LastComplianceReportAt sql.NullTime
	NextComplianceReportAt sql.NullTime
	DescribeSourceJobs     []DescribeSourceJob   `gorm:"foreignKey:SourceID;constraint:OnDelete:CASCADE;"`
	ComplianceReportJobs   []ComplianceReportJob `gorm:"foreignKey:SourceID;constraint:OnDelete:CASCADE;"`
}

type DescribeSourceJob struct {
	gorm.Model
	SourceID             uuid.UUID             // Not the primary key but should be a unique identifier
	DescribeResourceJobs []DescribeResourceJob `gorm:"foreignKey:ParentJobID;constraint:OnDelete:CASCADE;"`
	Status               api.DescribeSourceJobStatus
}

type ComplianceReportJob struct {
	gorm.Model
	SourceID        uuid.UUID // Not the primary key but should be a unique identifier
	ReportCreatedAt int64
	Status          api2.ComplianceReportJobStatus
	FailureMessage  string // Should be NULLSTRING
}

type Assignment struct {
	SourceID  uuid.UUID `gorm:"primarykey"`
	PolicyID  uuid.UUID `gorm:"primarykey"`
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DescribeResourceJob struct {
	gorm.Model
	ParentJobID    uint
	ResourceType   string
	Status         api.DescribeResourceJobStatus
	FailureMessage string // Should be NULLSTRING
}
