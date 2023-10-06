package describe

import (
	"sort"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/analytics"

	"github.com/kaytu-io/kaytu-engine/pkg/describe/enums"

	insightapi "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/lib/pq"

	checkupapi "github.com/kaytu-io/kaytu-engine/pkg/checkup/api"
	summarizerapi "github.com/kaytu-io/kaytu-engine/pkg/summarizer/api"

	api2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"

	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"gorm.io/gorm"
)

const (
	KaytuPrivateTagPrefix = "x-kaytu-"
	KaytuServiceCostTag   = KaytuPrivateTagPrefix + "cost-service-map"
)

func trimPrivateTags(tags map[string][]string) map[string][]string {
	for k := range tags {
		if strings.HasPrefix(k, KaytuPrivateTagPrefix) {
			delete(tags, k)
		}
	}
	return tags
}

type ComplianceReportJob struct {
	gorm.Model
	SourceID        string                         `json:"SourceId" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Not the primary key but should be a unique identifier
	SourceType      source.Type                    `json:"SourceType" example:"Azure"`
	BenchmarkID     string                         `json:"BenchmarkId" example:"azure_cis_v1"` // Not the primary key but should be a unique identifier
	ReportCreatedAt int64                          `json:"ReportCreatedAt" example:"1619510400"`
	Status          api2.ComplianceReportJobStatus `json:"Status" example:"InProgress"`
	FailureMessage  string                         // Should be NULLSTRING
	IsStack         bool                           `json:"IsStack" example:"false"`
}

type JobSequencerStatus string

const (
	JobSequencerWaitingForDependencies JobSequencerStatus = "WaitingForDependencies"
	JobSequencerFinished               JobSequencerStatus = "FINISHED"
	JobSequencerFailed                 JobSequencerStatus = "Failed"
)

type JobSequencerJobType string

const (
	JobSequencerJobTypeBenchmark           JobSequencerJobType = "Benchmark"
	JobSequencerJobTypeBenchmarkSummarizer JobSequencerJobType = "BenchmarkSummarizer"
	JobSequencerJobTypeDescribe            JobSequencerJobType = "Describe"
	JobSequencerJobTypeAnalytics           JobSequencerJobType = "Analytics"
)

type JobSequencer struct {
	gorm.Model
	DependencyList   pq.Int64Array `gorm:"type:bigint[]"`
	DependencySource string
	NextJob          string
	Status           JobSequencerStatus
}

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

type InsightJob struct {
	gorm.Model
	InsightID      uint   `gorm:"index:idx_source_id_insight_id"`
	SourceID       string `gorm:"index:idx_source_id_insight_id"`
	AccountID      string
	SourceType     source.Type
	Status         insightapi.InsightJobStatus
	FailureMessage string
	IsStack        bool
}

type SummarizerJob struct {
	gorm.Model
	ScheduleJobID  *uint
	Status         summarizerapi.SummarizerJobStatus
	JobType        summarizer.JobType
	FailureMessage string
}

type AnalyticsJob struct {
	gorm.Model
	ResourceCollectionId *string
	Status               analytics.JobStatus
	FailureMessage       string
}

type CheckupJob struct {
	gorm.Model
	Status         checkupapi.CheckupJobStatus
	FailureMessage string
}

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
