package inventory

import (
	"time"

	"github.com/opengovern/og-util/pkg/integration"

	"github.com/jackc/pgtype"
	"github.com/lib/pq"
	"github.com/opengovern/og-util/pkg/model"
	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/opencomply/services/inventory/api"
	"gorm.io/gorm"
)

type ResourceTypeTag struct {
	model.Tag
	ResourceType string `gorm:"primaryKey; type:citext"`
}

type NamedQueryTag struct {
	model.Tag
	NamedQueryID string `gorm:"primaryKey"`
}

type NamedQueryTagsResult struct {
	Key          string
	UniqueValues pq.StringArray `gorm:"type:text[]"`
}

func (s NamedQueryTagsResult) ToApi() api.NamedQueryTagsResult {
	return api.NamedQueryTagsResult{
		Key:          s.Key,
		UniqueValues: s.UniqueValues,
	}
}

type NamedQuery struct {
	ID               string         `gorm:"primarykey"`
	IntegrationTypes pq.StringArray `gorm:"type:text[]"`
	Title            string
	Description      string
	QueryID          *string
	Query            *Query `gorm:"foreignKey:QueryID;references:ID;constraint:OnDelete:SET NULL"`
	IsBookmarked     bool
	Tags             []NamedQueryTag `gorm:"foreignKey:NamedQueryID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type QueryParameter struct {
	QueryID  string `gorm:"primaryKey"`
	Key      string `gorm:"primaryKey"`
	Required bool   `gorm:"not null"`
}

func (qp QueryParameter) ToApi() api.QueryParameter {
	return api.QueryParameter{
		Key:      qp.Key,
		Required: qp.Required,
	}
}

type Query struct {
	ID             string `gorm:"primaryKey"`
	QueryToExecute string
	PrimaryTable   *string
	ListOfTables   pq.StringArray `gorm:"type:text[]"`
	Engine         string
	NamedQuery     []NamedQuery     `gorm:"foreignKey:QueryID"`
	Parameters     []QueryParameter `gorm:"foreignKey:QueryID"`
	Global         bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (q Query) ToApi() api.Query {
	query := api.Query{
		ID:             q.ID,
		QueryToExecute: q.QueryToExecute,
		ListOfTables:   q.ListOfTables,
		PrimaryTable:   q.PrimaryTable,
		Engine:         q.Engine,
		Parameters:     make([]api.QueryParameter, 0, len(q.Parameters)),
		Global:         q.Global,
		CreatedAt:      q.CreatedAt,
		UpdatedAt:      q.UpdatedAt,
	}
	for _, p := range q.Parameters {
		query.Parameters = append(query.Parameters, p.ToApi())
	}
	return query
}

func (p NamedQuery) GetTagsMap() map[string][]string {
	var tagsMap map[string][]string
	if p.Tags != nil {
		tagLikeArr := make([]model.TagLike, 0, len(p.Tags))
		for _, tag := range p.Tags {
			tagLikeArr = append(tagLikeArr, tag)
		}
		tagsMap = model.GetTagsMap(tagLikeArr)
	}
	return tagsMap
}

type NamedQueryHistory struct {
	Query      string `gorm:"type:citext; primaryKey"`
	ExecutedAt time.Time
}

func (s NamedQueryHistory) ToApi() api.NamedQueryHistory {
	return api.NamedQueryHistory{
		Query:      s.Query,
		ExecutedAt: s.ExecutedAt,
	}
}

type ResourceType struct {
	IntegrationType integration.Type `json:"integration_type" gorm:"index"`
	ResourceType    string           `json:"resource_type" gorm:"primaryKey; type:citext"`
	ResourceLabel   string           `json:"resource_name"`
	ServiceName     string           `json:"service_name" gorm:"index"`
	DoSummarize     bool             `json:"do_summarize"`
	LogoURI         *string          `json:"logo_uri,omitempty"`

	Tags    []ResourceTypeTag   `gorm:"foreignKey:ResourceType;references:ResourceType;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap map[string][]string `gorm:"-:all"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (r ResourceType) ToApi() api.ResourceType {
	apiResourceType := api.ResourceType{
		IntegrationType: r.IntegrationType,
		ResourceType:    r.ResourceType,
		ResourceLabel:   r.ResourceLabel,
		ServiceName:     r.ServiceName,
		Tags:            model.TrimPrivateTags(r.GetTagsMap()),
		LogoURI:         r.LogoURI,
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

type ResourceCollectionStatus string

const (
	ResourceCollectionStatusActive   ResourceCollectionStatus = "active"
	ResourceCollectionStatusInactive ResourceCollectionStatus = "inactive"
)

func (r ResourceCollectionStatus) ToApi() api.ResourceCollectionStatus {
	switch r {
	case ResourceCollectionStatusActive:
		return api.ResourceCollectionStatusActive
	case ResourceCollectionStatusInactive:
		return api.ResourceCollectionStatusInactive
	default:
		return api.ResourceCollectionStatusUnknown
	}
}

type ResourceCollection struct {
	ID          string `gorm:"primarykey"`
	Name        string
	FiltersJson pgtype.JSONB `gorm:"type:jsonb"`
	Description string
	Status      ResourceCollectionStatus

	Tags    []ResourceCollectionTag `gorm:"foreignKey:ResourceCollectionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap map[string][]string     `gorm:"-:all"`

	Created   time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Filters []opengovernance.ResourceCollectionFilter `gorm:"-:all"`
}

func (r ResourceCollection) ToApi() api.ResourceCollection {
	apiResourceCollection := api.ResourceCollection{
		ID:          r.ID,
		Name:        r.Name,
		Tags:        model.TrimPrivateTags(r.GetTagsMap()),
		Description: r.Description,
		CreatedAt:   r.Created,
		Status:      r.Status.ToApi(),
		Filters:     r.Filters,
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

type ResourceTypeV2 struct {
	IntegrationType integration.Type `gorm:"column:integration_type"`
	ResourceName    string           `gorm:"column:resource_name"`
	ResourceID      string           `gorm:"primaryKey"`
	SteampipeTable  string           `gorm:"column:steampipe_table"`
	Category        string           `gorm:"column:category"`
}

func (r ResourceTypeV2) ToApi() api.ResourceTypeV2 {
	apiResourceType := api.ResourceTypeV2{
		IntegrationType: r.IntegrationType,
		ResourceName:    r.ResourceName,
		ResourceID:      r.ResourceID,
		SteampipeTable:  r.SteampipeTable,
		Category:        r.Category,
	}
	return apiResourceType
}

type CategoriesTables struct {
	Category string   `json:"category"`
	Tables   []string `json:"tables"`
}

func (r CategoriesTables) ToApi() api.CategoriesTables {
	apiResourceType := api.CategoriesTables{
		Tables:   r.Tables,
		Category: r.Category,
	}
	return apiResourceType
}
