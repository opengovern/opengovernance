package model

import (
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"sort"
	"strings"
	"time"
)

const (
	KaytuPrivateTagPrefix = "x-kaytu-"
	KaytuServiceCostTag   = KaytuPrivateTagPrefix + "cost-service-map"
)

type Stack struct {
	StackID        string         `gorm:"primarykey"`
	Resources      pq.StringArray `gorm:"type:text[]"`
	AccountIDs     pq.StringArray `gorm:"type:text[]"`
	SourceType     source.Type    `gorm:"type:text"`
	ResourceTypes  pq.StringArray `gorm:"type:text[]"`
	Status         api.StackStatus
	FailureMessage string

	Evaluations []*StackEvaluation  `gorm:"foreignKey:StackID;references:StackID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Tags        []*StackTag         `gorm:"foreignKey:StackID;references:StackID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap     map[string][]string `gorm:"-:all"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (s Stack) ToApi() api.Stack {
	var evaluations []api.StackEvaluation
	for _, e := range s.Evaluations {
		evaluations = append(evaluations, api.StackEvaluation{
			Type:        e.Type,
			EvaluatorID: e.EvaluatorID,
			JobID:       e.JobID,
			CreatedAt:   e.CreatedAt,
			Status:      e.Status,
		})
	}

	stack := api.Stack{
		StackID:        s.StackID,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
		Resources:      []string(s.Resources),
		ResourceTypes:  []string(s.ResourceTypes),
		Tags:           trimPrivateTags(s.GetTagsMap()),
		Evaluations:    evaluations,
		AccountIDs:     s.AccountIDs,
		SourceType:     s.SourceType,
		Status:         s.Status,
		FailureMessage: s.FailureMessage,
	}
	return stack
}

type StackTag struct {
	Key     string         `gorm:"primaryKey;index:idx_key;index:idx_key_value"`
	Value   pq.StringArray `gorm:"type:text[];index:idx_key_value"`
	StackID string         `gorm:"primaryKey"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type StackEvaluation struct {
	EvaluatorID string
	Type        api.EvaluationType
	StackID     string
	JobID       uint `gorm:"primaryKey"`
	Status      api.StackEvaluationStatus

	CreatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type StackCredential struct {
	StackID string `gorm:"primarykey"`
	Secret  string
}

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
		sort.Slice(result[k], func(i, j int) bool {
			return result[k][i] < result[k][j]
		})
	}

	return result
}

func (t StackTag) GetKey() string {
	return t.Key
}

func (t StackTag) GetValue() []string {
	return t.Value
}

func (r Stack) GetTagsMap() map[string][]string {
	if r.tagsMap == nil {
		tagLikeArr := make([]TagLike, 0, len(r.Tags))
		for _, tag := range r.Tags {
			tagLikeArr = append(tagLikeArr, tag)
		}
		r.tagsMap = getTagsMap(tagLikeArr)
	}
	return r.tagsMap
}

func trimPrivateTags(tags map[string][]string) map[string][]string {
	for k := range tags {
		if strings.HasPrefix(k, KaytuPrivateTagPrefix) {
			delete(tags, k)
		}
	}
	return tags
}
