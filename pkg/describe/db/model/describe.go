package model

import (
	"github.com/lib/pq"
	"github.com/opengovern/og-util/pkg/describe/enums"
	"github.com/opengovern/og-util/pkg/source"
	"github.com/opengovern/opengovernance/pkg/describe/api"
	"gorm.io/gorm"
	"time"
)

type DiscoveryType string
type IntegrationDiscoveryStatus string

const (
	DiscoveryType_Fast DiscoveryType = "FAST"
	DiscoveryType_Full DiscoveryType = "FULL"
	DiscoveryType_Cost DiscoveryType = "COST"

	IntegrationDiscoveryStatusInProgress IntegrationDiscoveryStatus = "IN_PROGRESS"
	IntegrationDiscoveryStatusCompleted  IntegrationDiscoveryStatus = "COMPLETED"
)

type IntegrationDiscovery struct {
	ID            uint `gorm:"primaryKey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	TriggerID     uint
	ConnectionID  string                    `json:"connectionID"`
	AccountID     string                    `json:"accountID"`
	TriggerType   enums.DescribeTriggerType `json:"triggerType"`
	TriggeredBy   string                    `json:"triggeredBy"`
	DiscoveryType DiscoveryType
	ResourceTypes pq.StringArray `gorm:"type:text[]" json:"resourceTypes"`
}

type DescribeConnectionJob struct {
	ID             uint `gorm:"primarykey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time      `gorm:"index:,sort:desc"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	QueuedAt       time.Time
	InProgressedAt time.Time
	CreatedBy      string
	ParentID       *uint `gorm:"index:,sort:desc"`

	ConnectionID string `gorm:"index:idx_source_id_full_discovery;index"`
	Connector    source.Type
	AccountID    string
	TriggerType  enums.DescribeTriggerType

	ResourceType           string `gorm:"index:idx_resource_type_status;index"`
	DiscoveryType          DiscoveryType
	Status                 api.DescribeResourceJobStatus `gorm:"index:idx_resource_type_status;index"`
	RetryCount             int
	FailureMessage         string // Should be NULLSTRING
	ErrorCode              string // Should be NULLSTRING
	DescribedResourceCount int64
	DeletingCount          int64

	NatsSequenceNumber uint64
}
