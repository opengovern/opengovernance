package inventory

import (
	"github.com/jackc/pgtype"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/lib/pq"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/gorm"
)

type ResourceTypeTag struct {
	model.Tag
	ResourceType string `gorm:"primaryKey; type:citext"`
}

type SmartQuery struct {
	ID         string         `gorm:"primarykey"`
	Connectors pq.StringArray `gorm:"type:text[]"`
	Title      string
	Query      string
	IsPopular  bool
}

type SmartQueryHistory struct {
	Query      string `gorm:"type:citext; primaryKey"`
	ExecutedAt time.Time
}

func (s SmartQueryHistory) ToApi() api.SmartQueryHistory {
	return api.SmartQueryHistory{
		Query:      s.Query,
		ExecutedAt: s.ExecutedAt,
	}
}

type ResourceType struct {
	Connector     source.Type `json:"connector" gorm:"index"`
	ResourceType  string      `json:"resource_type" gorm:"primaryKey; type:citext"`
	ResourceLabel string      `json:"resource_name"`
	ServiceName   string      `json:"service_name" gorm:"index"`
	DoSummarize   bool        `json:"do_summarize"`
	LogoURI       *string     `json:"logo_uri,omitempty"`

	Tags    []ResourceTypeTag   `gorm:"foreignKey:ResourceType;references:ResourceType;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap map[string][]string `gorm:"-:all"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (r ResourceType) ToApi() api.ResourceType {
	apiResourceType := api.ResourceType{
		Connector:     r.Connector,
		ResourceType:  r.ResourceType,
		ResourceLabel: r.ResourceLabel,
		ServiceName:   r.ServiceName,
		Tags:          model.TrimPrivateTags(r.GetTagsMap()),
		LogoURI:       r.LogoURI,
	}
	return apiResourceType
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

type ResourceCollectionTag struct {
	model.Tag
	ResourceCollectionID string `gorm:"primaryKey"`
}

type ResourceCollection struct {
	ID          string `gorm:"primarykey"`
	Name        string
	FiltersJson pgtype.JSONB `gorm:"type:jsonb"`

	Tags    []ResourceCollectionTag `gorm:"foreignKey:ResourceCollectionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap map[string][]string     `gorm:"-:all"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Filters []kaytu.ResourceCollectionFilter `gorm:"-:all"`
}

func (r ResourceCollection) ToApi() api.ResourceCollection {
	apiResourceCollection := api.ResourceCollection{
		ID:      r.ID,
		Name:    r.Name,
		Tags:    model.TrimPrivateTags(r.GetTagsMap()),
		Filters: r.Filters,
	}
	return apiResourceCollection
}

func (r ResourceCollection) GetTagsMap() map[string][]string {
	if r.tagsMap == nil {
		tagLikeArr := make([]model.TagLike, 0, len(r.Tags))
		for _, tag := range r.Tags {
			tagLikeArr = append(tagLikeArr, tag)
		}
		r.tagsMap = model.GetTagsMap(tagLikeArr)
	}
	return r.tagsMap
}
