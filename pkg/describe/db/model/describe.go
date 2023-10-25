package model

import (
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/enums"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/gorm"
	"time"
)

type DescribeConnectionJob struct {
	ID             uint `gorm:"primarykey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time      `gorm:"index:,sort:desc"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	QueuedAt       time.Time
	InProgressedAt time.Time

	ConnectionID string `gorm:"index:idx_source_id_full_discovery;index"`
	Connector    source.Type
	AccountID    string
	TriggerType  enums.DescribeTriggerType

	ResourceType           string                        `gorm:"index:idx_resource_type_status;index"`
	Status                 api.DescribeResourceJobStatus `gorm:"index:idx_resource_type_status;index"`
	RetryCount             int
	FailureMessage         string // Should be NULLSTRING
	ErrorCode              string // Should be NULLSTRING
	DescribedResourceCount int64
}
