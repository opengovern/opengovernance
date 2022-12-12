package inventory

import (
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

type Tag struct {
	gorm.Model
	Key          string
	Value        string
	SmartQueries []SmartQuery `gorm:"many2many:smartquery_tags;"`
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
	LastDayCount     *int
	LastWeekCount    *int
	LastQuarterCount *int
	LastYearCount    *int
	Count            int
}

type MetricHistory struct {
	SourceID      string `gorm:"primaryKey"`
	Provider      string `gorm:"index;index:provider_resource_type_date_idx"`
	ResourceType  string `gorm:"primaryKey;index:provider_resource_type_date_idx"`
	Date          int64  `gorm:"primaryKey;index:,sort:desc;index:provider_resource_type_date_idx"`
	ScheduleJobID uint
	Count         int
}

type MetricResourceTypeSummary struct {
	ResourceType     string
	Count            int
	LastDayCount     *int
	LastWeekCount    *int
	LastQuarterCount *int
	LastYearCount    *int
}
