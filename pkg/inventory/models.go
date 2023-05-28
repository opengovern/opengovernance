package inventory

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type TagLike interface {
	GetKey() string
	GetValue() []string
}

func getTagsMap(tags []TagLike) map[string][]string {
	tagsMapToMap := make(map[string]map[string]bool)
	for _, tag := range tags {
		if v, ok := tagsMapToMap[tag.GetKey()]; !ok {
			uniqueMap := make(map[string]bool)
			for _, val := range tag.GetValue() {
				uniqueMap[val] = true
			}
			tagsMapToMap[tag.GetKey()] = uniqueMap

		} else {
			for _, val := range tag.GetValue() {
				v[val] = true
			}
			tagsMapToMap[tag.GetKey()] = v
		}
	}

	result := make(map[string][]string)
	for k, v := range tagsMapToMap {
		for val := range v {
			result[k] = append(result[k], val)
		}
	}

	return result
}

type Tag struct {
	Key   string         `gorm:"primaryKey;index:idx_key;index:idx_key_value"`
	Value pq.StringArray `gorm:"type:text[];index:idx_key_value"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (t Tag) GetKey() string {
	return t.Key
}

func (t Tag) GetValue() []string {
	return t.Value
}

type ResourceTypeTag struct {
	Tag
	ResourceType string `gorm:"primaryKey"`
}

func (t ResourceTypeTag) GetKey() string {
	return t.Tag.Key
}

func (t ResourceTypeTag) GetValue() []string {
	return t.Tag.Value
}

type ServiceTag struct {
	Tag
	ServiceName string `gorm:"primaryKey"`
}

func (t ServiceTag) GetKey() string {
	return t.Tag.Key
}

func (t ServiceTag) GetValue() []string {
	return t.Tag.Value
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
	TagsMap map[string][]string `gorm:"-:all"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (r ResourceType) GetTagsMap() map[string][]string {
	if r.TagsMap == nil {
		tagLikeArr := make([]TagLike, 0, len(r.Tags))
		for _, tag := range r.Tags {
			tagLikeArr = append(tagLikeArr, tag)
		}
		r.TagsMap = getTagsMap(tagLikeArr)
	}
	return r.TagsMap
}

type Service struct {
	ServiceName   string         `json:"service_name" gorm:"primaryKey"`
	ServiceLabel  string         `json:"service_label"`
	Connector     source.Type    `json:"connector" gorm:"index"`
	LogoURI       *string        `json:"logo_uri,omitempty"`
	ResourceTypes []ResourceType `json:"resource_types" gorm:"foreignKey:ServiceName;references:ServiceName;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	Tags    []ServiceTag        `gorm:"foreignKey:ServiceName;references:ServiceName;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	TagsMap map[string][]string `gorm:"-:all"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (s Service) GetTagsMap() map[string][]string {
	if s.TagsMap == nil {
		tagLikeArr := make([]TagLike, 0, len(s.Tags))
		for _, tag := range s.Tags {
			tagLikeArr = append(tagLikeArr, tag)
		}
		s.TagsMap = getTagsMap(tagLikeArr)
	}
	return s.TagsMap
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
