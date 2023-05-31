package inventory

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/gorm"
)

type ResourceTypeTag struct {
	model.Tag
	ResourceType string `gorm:"primaryKey"`
}

type ServiceTag struct {
	model.Tag
	ServiceName string `gorm:"primaryKey"`
}

type SmartQuery struct {
	gorm.Model
	Provider    string
	Title       string
	Description string
	Query       string
}

type ResourceType struct {
	Connector     source.Type `json:"connector" gorm:"index"`
	ResourceType  string      `json:"resource_type" gorm:"primaryKey"`
	ResourceLabel string      `json:"resource_name"`
	ServiceName   string      `json:"service_name" gorm:"index"`
	LogoURI       *string     `json:"logo_uri,omitempty"`

	Tags    []ResourceTypeTag   `gorm:"foreignKey:ResourceType;references:ResourceType;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap map[string][]string `gorm:"-:all"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (r ResourceType) GetTagsMap() map[string][]string {
	if r.tagsMap == nil {
		tagLikeArr := make([]model.TagLike, 0, len(r.Tags))
		for _, tag := range r.Tags {
			tagLikeArr = append(tagLikeArr, tag)
		}
		r.tagsMap = model.GetTagsMap(tagLikeArr)
	}
	return r.tagsMap
}

type Service struct {
	ServiceName   string         `json:"service_name" gorm:"primaryKey"`
	ServiceLabel  string         `json:"service_label"`
	Connector     source.Type    `json:"connector" gorm:"index"`
	LogoURI       *string        `json:"logo_uri,omitempty"`
	ResourceTypes []ResourceType `json:"resource_types" gorm:"foreignKey:ServiceName;references:ServiceName;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	Tags    []ServiceTag        `gorm:"foreignKey:ServiceName;references:ServiceName;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap map[string][]string `gorm:"-:all"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (s Service) GetTagsMap() map[string][]string {
	if s.tagsMap == nil {
		tagLikeArr := make([]model.TagLike, 0, len(s.Tags))
		for _, tag := range s.Tags {
			tagLikeArr = append(tagLikeArr, tag)
		}
		s.tagsMap = model.GetTagsMap(tagLikeArr)
	}
	return s.tagsMap
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
