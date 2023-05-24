package inventory

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/gorm"
)

type SmartQuery struct {
	gorm.Model
	Provider    string
	Title       string
	Description string
	Query       string
	Tags        []Tag `gorm:"many2many:smartquery_tags;"`
}

type ResourceType struct {
	Connector     source.Type `json:"connector" gorm:"index"`
	ResourceType  string      `json:"resource_type" gorm:"primaryKey"`
	ResourceLabel string      `json:"resource_name"`
	ServiceName   string      `json:"service_name" gorm:"index"`
	LogoURI       *string     `json:"logo_uri,omitempty"`

	Tags []Tag `gorm:"many2many:resource_type_tags;"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Service struct {
	ServiceName   string         `json:"service_name" gorm:"primaryKey"`
	ServiceLabel  string         `json:"service_label"`
	Connector     source.Type    `json:"connector" gorm:"index"`
	LogoURI       *string        `json:"logo_uri,omitempty"`
	ResourceTypes []ResourceType `json:"resource_types" gorm:"foreignKey:ServiceName,references:ServiceName;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	Tags []Tag `gorm:"many2many:service_tags;"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Tag struct {
	gorm.Model
	Key   string `gorm:"index:idx_key;index:idx_key_value"`
	Value string `gorm:"index:idx_key_value"`

	SmartQueries  []SmartQuery   `gorm:"many2many:smartquery_tags;"`
	ResourceTypes []ResourceType `gorm:"many2many:resource_type_tags;"`
	Services      []Service      `gorm:"many2many:service_tags;"`
}

type Category struct {
	Name         string `gorm:"primaryKey"`
	SubCategory  string `gorm:"primaryKey"`
	Cloud        string
	CloudService string `gorm:"primaryKey"`
}

type Metric struct {
	SourceID         string `gorm:"primaryKey"`
	Provider         string `gorm:"index"`
	ResourceType     string `gorm:"primaryKey"`
	ScheduleJobID    uint
	SummarizeJobID   *uint
	LastDayCount     *int
	LastWeekCount    *int
	LastQuarterCount *int
	LastYearCount    *int
	Count            int
}
