package describe

import (
	"database/sql"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
	"time"

	insightapi "gitlab.com/keibiengine/keibi-engine/pkg/insight/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer"

	checkupapi "gitlab.com/keibiengine/keibi-engine/pkg/checkup/api"
	summarizerapi "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/api"

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"

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

type ComplianceReportJob struct {
	gorm.Model
	ScheduleJobID   uint
	SourceID        string // Not the primary key but should be a unique identifier
	SourceType      source.Type
	BenchmarkID     string // Not the primary key but should be a unique identifier
	ReportCreatedAt int64
	Status          api2.ComplianceReportJobStatus
	FailureMessage  string // Should be NULLSTRING
}

type ScheduleJob struct {
	gorm.Model
	Status         summarizerapi.SummarizerJobStatus
	FailureMessage string
}

type DescribeSourceJob struct {
	gorm.Model
	DescribedAt          time.Time
	SourceID             uuid.UUID // Not the primary key but should be a unique identifier
	AccountID            string
	DescribeResourceJobs []DescribeResourceJob `gorm:"foreignKey:ParentJobID;constraint:OnDelete:CASCADE;"`
	Status               api.DescribeSourceJobStatus
	TriggerType          enums.DescribeTriggerType
}

type CloudNativeDescribeSourceJob struct {
	gorm.Model
	JobID                          uuid.UUID         `gorm:"type:uuid;default:uuid_generate_v4();uniqueIndex"`
	SourceJob                      DescribeSourceJob `gorm:"foreignKey:SourceJobID;references:ID;"`
	SourceJobID                    uint
	CredentialEncryptionPrivateKey string
	CredentialEncryptionPublicKey  string
	ResultEncryptionPrivateKey     string
	ResultEncryptionPublicKey      string
}

type DescribeResourceJob struct {
	gorm.Model
	ParentJobID    uint
	ResourceType   string
	Status         api.DescribeResourceJobStatus
	FailureMessage string // Should be NULLSTRING
}

type InsightJob struct {
	gorm.Model
	InsightID      uint
	SourceID       string
	AccountID      string
	ScheduleUUID   string
	SourceType     source.Type
	Status         insightapi.InsightJobStatus
	FailureMessage string
}

type SummarizerJob struct {
	gorm.Model
	ScheduleJobID  *uint
	Status         summarizerapi.SummarizerJobStatus
	JobType        summarizer.JobType
	FailureMessage string
}

type CheckupJob struct {
	gorm.Model
	Status         checkupapi.CheckupJobStatus
	FailureMessage string
}
