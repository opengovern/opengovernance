package db

import (
	"context"
	"fmt"
	kaytuTrace "github.com/kaytu-io/kaytu-util/pkg/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/lib/pq"

	"github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"

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

	Tags    []BenchmarkTag      `gorm:"foreignKey:BenchmarkID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap map[string][]string `gorm:"-:all"`

	Children  []Benchmark `gorm:"many2many:benchmark_children;"`
	Policies  []Policy    `gorm:"many2many:benchmark_policies;"`
	CreatedAt time.Time
	UpdatedAt time.Time
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
		Tags:        b.GetTagsMap(),
	}
	for _, child := range b.Children {
		ba.Children = append(ba.Children, child.ID)
	}
	for _, policy := range b.Policies {
		ba.Policies = append(ba.Policies, policy.ID)
	}
	return ba
}

func (b *Benchmark) PopulateConnectors(ctx context.Context, db Database, api *api.Benchmark) error {
	if len(api.Connectors) > 0 {
		return nil
	}
	// tracer :
	output2, span2 := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx, "new_GetBenchmark(loop)", trace.WithSpanKind(trace.SpanKindClient))
	span2.SetName("new_GetBenchmark(loop)")
	for _, childObj := range b.Children {
		//tracer :
		_, span3 := otel.Tracer(kaytuTrace.JaegerTracerName).Start(output2, "new_GetBenchmark", trace.WithSpanKind(trace.SpanKindClient))
		span3.SetName("new_GetBenchmark")

		child, err := db.GetBenchmark(childObj.ID)
		if err != nil {
			return err
		}
		span3.End()

		if child == nil {
			return fmt.Errorf("child %s not found", childObj.ID)
		}

		ca := child.ToApi()
		err = child.PopulateConnectors(ctx, db, &ca)
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
	span2.End()
	// tracer :
	output4, span4 := otel.Tracer(kaytuTrace.JaegerTracerName).Start(output2, "new_GetQuery(loop)", trace.WithSpanKind(trace.SpanKindClient))
	span4.SetName("new_GetQuery(loop)")

	for _, policy := range b.Policies {
		if policy.QueryID == nil {
			continue
		}
		//tracer :
		_, span5 := otel.Tracer(kaytuTrace.JaegerTracerName).Start(output4, "new_GetQuery", trace.WithSpanKind(trace.SpanKindClient))
		span5.SetName("new_GetQuery")

		query, err := db.GetQuery(*policy.QueryID)
		if err != nil {
			return err
		}
		span5.End()

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
	span4.End()

	return nil
}

func (b Benchmark) GetTagsMap() map[string][]string {
	if b.tagsMap == nil {
		tagLikeArr := make([]model.TagLike, 0, len(b.Tags))
		for _, tag := range b.Tags {
			tagLikeArr = append(tagLikeArr, tag)
		}
		b.tagsMap = model.GetTagsMap(tagLikeArr)
	}
	return b.tagsMap
}

type BenchmarkChild struct {
	BenchmarkID string
	ChildID     string
}

type BenchmarkTag struct {
	model.Tag
	BenchmarkID string `gorm:"primaryKey"`
}

type Policy struct {
	ID          string `gorm:"primaryKey"`
	Title       string
	Description string

	Tags    []PolicyTag         `gorm:"foreignKey:PolicyID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap map[string][]string `gorm:"-:all"`

	DocumentURI        string
	Enabled            bool
	QueryID            *string
	Benchmarks         []Benchmark `gorm:"many2many:benchmark_policies;"`
	Severity           types.FindingSeverity
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
		Tags:               p.GetTagsMap(),
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
	return pa
}

func (p Policy) GetTagsMap() map[string][]string {
	if p.tagsMap == nil {
		tagLikeArr := make([]model.TagLike, 0, len(p.Tags))
		for _, tag := range p.Tags {
			tagLikeArr = append(tagLikeArr, tag)
		}
		p.tagsMap = model.GetTagsMap(tagLikeArr)
	}
	return p.tagsMap
}

func (p *Policy) PopulateConnector(ctx context.Context, db Database, api *api.Policy) error {
	tracer := otel.Tracer("PopulateConnector")
	if !api.Connector.IsNull() {
		return nil
	}

	if p.QueryID == nil {
		return nil
	}
	// tracer :
	_, span1 := tracer.Start(ctx, "new_GetQuery", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetQuery")

	query, err := db.GetQuery(*p.QueryID)
	if err != nil {
		return err
	}
	span1.End()

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
	model.Tag
	PolicyID string `gorm:"primaryKey"`
}

type BenchmarkPolicies struct {
	BenchmarkID string
	PolicyID    string
}

type Insight struct {
	gorm.Model
	QueryID     string
	Query       Query `gorm:"foreignKey:QueryID;references:ID;constraint:OnDelete:CASCADE;"`
	Connector   source.Type
	ShortTitle  string
	LongTitle   string
	Description string
	LogoURL     *string

	Tags    []InsightTag        `gorm:"foreignKey:InsightID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap map[string][]string `gorm:"-:all"`

	Links    pq.StringArray `gorm:"type:text[]"`
	Enabled  bool
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

type InsightGroup struct {
	gorm.Model
	ShortTitle  string
	LongTitle   string
	Description string
	LogoURL     *string

	Insights []Insight `gorm:"many2many:insight_group_insights;"`
}

func (i InsightGroup) ToApi() api.InsightGroup {
	ia := api.InsightGroup{
		ID:          i.ID,
		ShortTitle:  i.ShortTitle,
		LongTitle:   i.LongTitle,
		Description: i.Description,
		LogoURL:     i.LogoURL,
		Insights:    nil,
	}
	connectorsMap := make(map[source.Type]bool)
	for _, insight := range i.Insights {
		ia.Insights = append(ia.Insights, insight.ToApi())
		connectorsMap[insight.Connector] = true
	}
	ia.Connectors = make([]source.Type, 0, len(connectorsMap))
	for connector := range connectorsMap {
		ia.Connectors = append(ia.Connectors, connector)
	}
	tags := make([]model.TagLike, 0)
	for _, insight := range i.Insights {
		for _, v := range insight.Tags {
			tags = append(tags, v)
		}
	}
	ia.Tags = model.GetTagsMap(tags)

	return ia
}

type InsightGroupInsight struct {
	InsightGroupID uint `gorm:"primaryKey"`
	InsightID      uint `gorm:"primaryKey"`
}

type Query struct {
	ID             string `gorm:"primaryKey"`
	QueryToExecute string
	Connector      string
	ListOfTables   pq.StringArray `gorm:"type:text[]"`
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
