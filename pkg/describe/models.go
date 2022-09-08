package describe

import (
	"database/sql"
	"time"

	summarizerapi "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/api"

	insightapi "gitlab.com/keibiengine/keibi-engine/pkg/insight/api"

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/api"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gorm.io/gorm"
)

type Source struct {
	ID                     uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	AccountID              string
	Type                   api.SourceType
	ConfigRef              string
	LastDescribedAt        sql.NullTime
	NextDescribeAt         sql.NullTime
	LastComplianceReportAt sql.NullTime
	NextComplianceReportAt sql.NullTime
	DescribeSourceJobs     []DescribeSourceJob   `gorm:"foreignKey:SourceID;constraint:OnDelete:CASCADE;"`
	ComplianceReportJobs   []ComplianceReportJob `gorm:"foreignKey:SourceID;constraint:OnDelete:CASCADE;"`
	NextComplianceReportID uint                  `gorm:"default:0"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}

type DescribeSourceJob struct {
	gorm.Model
	SourceID             uuid.UUID // Not the primary key but should be a unique identifier
	AccountID            string
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

type DescribeResourceJob struct {
	gorm.Model
	ParentJobID    uint
	ResourceType   string
	Status         api.DescribeResourceJobStatus
	FailureMessage string // Should be NULLSTRING
}

type Insight struct {
	gorm.Model
	Description  string
	Query        string
	SmartQueryID uint
	Internal     bool
	Provider     string
	Category     string
}

type InsightJob struct {
	gorm.Model
	InsightID      uint
	Status         insightapi.InsightJobStatus
	FailureMessage string
}

type SummarizerJob struct {
	gorm.Model
	SourceID       uuid.UUID
	SourceJobID    uint
	Status         summarizerapi.SummarizerJobStatus
	FailureMessage string
}
