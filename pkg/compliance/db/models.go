package db

import (
	"fmt"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/lib/pq"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"

	"gorm.io/gorm"
)

type BenchmarkAssignment struct {
	gorm.Model
	BenchmarkId  string `gorm:"index:idx_benchmark_source"`
	ConnectionId string `gorm:"index:idx_benchmark_source"`
	AssignedAt   time.Time
}

type Benchmark struct {
	ID          string `gorm:"primarykey"`
	Title       string
	Description string
	LogoURI     string
	Category    string
	DocumentURI string
	Enabled     bool
	Managed     bool
	AutoAssign  bool
	Baseline    bool
	Tags        []BenchmarkTag `gorm:"many2many:benchmark_tag_rels;"`
	Children    []Benchmark    `gorm:"many2many:benchmark_children;"`
	Policies    []Policy       `gorm:"many2many:benchmark_policies;"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (b Benchmark) ToApi() api.Benchmark {
	ba := api.Benchmark{
		ID:          b.ID,
		Title:       b.Title,
		Description: b.Description,
		LogoURI:     b.LogoURI,
		Category:    b.Category,
		DocumentURI: b.DocumentURI,
		Enabled:     b.Enabled,
		Managed:     b.Managed,
		AutoAssign:  b.AutoAssign,
		Baseline:    b.Baseline,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}
	ba.Tags = map[string]string{}
	for _, tag := range b.Tags {
		ba.Tags[tag.Key] = tag.Value
	}
	for _, child := range b.Children {
		ba.Children = append(ba.Children, child.ID)
	}
	for _, policy := range b.Policies {
		ba.Policies = append(ba.Policies, policy.ID)
	}
	return ba
}

func (b *Benchmark) PopulateConnectors(db Database, api *api.Benchmark) error {
	if len(api.Connectors) > 0 {
		return nil
	}

	for _, childObj := range b.Children {
		child, err := db.GetBenchmark(childObj.ID)
		if err != nil {
			return err
		}
		if child == nil {
			return fmt.Errorf("child %s not found", childObj.ID)
		}

		ca := child.ToApi()
		err = child.PopulateConnectors(db, &ca)
		if err != nil {
			return err
		}

		for _, conn := range ca.Connectors {

			exists := false
			for _, c := range api.Connectors {
				if c == conn {
					exists = true
				}
			}
			if !exists {
				api.Connectors = append(api.Connectors, conn)
			}
		}
	}

	for _, policy := range b.Policies {
		query, err := db.GetQuery(*policy.QueryID)
		if err != nil {
			return err
		}
		if query == nil {
			return fmt.Errorf("query %s not found", *policy.QueryID)
		}

		ty, err := source.ParseType(query.Connector)
		if err != nil {
			return err
		}

		exists := false
		for _, c := range api.Connectors {
			if c == ty {
				exists = true
			}
		}
		if !exists {
			api.Connectors = append(api.Connectors, ty)
		}
	}

	return nil
}

type BenchmarkChild struct {
	BenchmarkID string
	ChildID     string
}

type BenchmarkTag struct {
	gorm.Model
	Key        string
	Value      string
	Benchmarks []Benchmark `gorm:"many2many:benchmark_tag_rels;"`
}

type BenchmarkTagRel struct {
	BenchmarkID    string
	BenchmarkTagID uint
}

type Policy struct {
	ID                 string `gorm:"primarykey"`
	Title              string
	Description        string
	Tags               []PolicyTag `gorm:"many2many:policy_tag_rels;"`
	DocumentURI        string
	Enabled            bool
	QueryID            *string
	Benchmarks         []Benchmark `gorm:"many2many:benchmark_policies;"`
	Severity           types.Severity
	ManualVerification bool
	Managed            bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (p Policy) ToApi() api.Policy {
	pa := api.Policy{
		ID:                 p.ID,
		Title:              p.Title,
		Description:        p.Description,
		Tags:               nil,
		Connector:          "",
		Enabled:            p.Enabled,
		DocumentURI:        p.DocumentURI,
		QueryID:            p.QueryID,
		Severity:           p.Severity,
		ManualVerification: p.ManualVerification,
		Managed:            p.Managed,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
	pa.Tags = map[string]string{}
	for _, tag := range p.Tags {
		pa.Tags[tag.Key] = tag.Value
	}
	return pa
}

func (p *Policy) PopulateConnector(db Database, api *api.Policy) error {
	if !api.Connector.IsNull() {
		return nil
	}

	query, err := db.GetQuery(*p.QueryID)
	if err != nil {
		return err
	}
	if query == nil {
		return fmt.Errorf("query %s not found", *p.QueryID)
	}

	ty, err := source.ParseType(query.Connector)
	if err != nil {
		return err
	}

	api.Connector = ty
	return nil
}

type PolicyTag struct {
	gorm.Model
	Key      string
	Value    string
	Policies []Policy `gorm:"many2many:policy_tag_rels;"`
}

type PolicyTagRel struct {
	PolicyID    string
	PolicyTagID uint
}

type BenchmarkPolicies struct {
	BenchmarkID string
	PolicyID    string
}

type Insight struct {
	gorm.Model
	QueryID     string
	Query       Query `gorm:"foreignKey:QueryID;references:ID;constraint:OnDelete:CASCADE;"`
	ShortTitle  string
	LongTitle   string
	Description string
	LogoURL     *string

	Tags    []InsightTag        `gorm:"foreignKey:InsightID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap map[string][]string `gorm:"-:all"`

	Links    pq.StringArray `gorm:"type:text[]"`
	Enabled  bool           `gorm:"default:true"`
	Internal bool
}

func (i Insight) GetTagsMap() map[string][]string {
	if i.tagsMap == nil {
		tagLikeArr := make([]model.TagLike, 0, len(i.Tags))
		for _, tag := range i.Tags {
			tagLikeArr = append(tagLikeArr, tag)
		}
		i.tagsMap = model.GetTagsMap(tagLikeArr)
	}
	return i.tagsMap
}

func (i Insight) ToApi() api.Insight {
	ia := api.Insight{
		ID:          i.ID,
		PeerGroupId: i.PeerGroupId,
		Query:       i.Query.ToApi(),
		Connector:   i.Connector,
		ShortTitle:  i.ShortTitle,
		LongTitle:   i.LongTitle,
		Description: i.Description,
		LogoURL:     i.LogoURL,
		Tags:        i.GetTagsMap(),
		Links:       i.Links,
		Enabled:     i.Enabled,
		Internal:    i.Internal,
	}

	return ia
}

type InsightTag struct {
	model.Tag
	InsightID uint `gorm:"primaryKey"`
}

type Query struct {
	ID             string `gorm:"primaryKey"`
	QueryToExecute string
	Connector      string
	ListOfTables   string
	Engine         string
	Policies       []Policy  `gorm:"foreignKey:QueryID"`
	Insights       []Insight `gorm:"foreignKey:QueryID"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (q Query) ToApi() api.Query {
	return api.Query{
		ID:             q.ID,
		QueryToExecute: q.QueryToExecute,
		Connector:      q.Connector,
		ListOfTables:   q.ListOfTables,
		Engine:         q.Engine,
		CreatedAt:      q.CreatedAt,
		UpdatedAt:      q.UpdatedAt,
	}
}
