package db

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	BenchmarkId        string  `gorm:"index:idx_benchmark_source; index:idx_benchmark_rc; not null"`
	ConnectionId       *string `gorm:"index:idx_benchmark_source"`
	ResourceCollection *string `gorm:"index:idx_benchmark_rc"`
	AssignedAt         time.Time
}

type Benchmark struct {
	ID          string `gorm:"primarykey"`
	Title       string
	DisplayCode string
	Connector   source.Type
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
	Controls  []Control   `gorm:"many2many:benchmark_controls;"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (b Benchmark) ToApi() api.Benchmark {
	ba := api.Benchmark{
		ID:          b.ID,
		Title:       b.Title,
		DisplayCode: b.DisplayCode,
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
	if b.Connector != source.Nil {
		ba.Connectors = []source.Type{b.Connector}
	}
	for _, child := range b.Children {
		ba.Children = append(ba.Children, child.ID)
	}
	for _, control := range b.Controls {
		ba.Controls = append(ba.Controls, control.ID)
	}
	return ba
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

type Control struct {
	ID          string `gorm:"primaryKey"`
	Title       string
	Description string

	Tags    []ControlTag        `gorm:"foreignKey:ControlID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap map[string][]string `gorm:"-:all"`

	DocumentURI        string
	Enabled            bool
	QueryID            *string
	Benchmarks         []Benchmark `gorm:"many2many:benchmark_controls;"`
	Severity           types.FindingSeverity
	ManualVerification bool
	Managed            bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (p Control) ToApi() api.Control {
	pa := api.Control{
		ID:                 p.ID,
		Title:              p.Title,
		Description:        p.Description,
		Tags:               model.TrimPrivateTags(p.GetTagsMap()),
		Explanation:        "",
		NonComplianceCost:  "",
		UsefulExample:      "",
		Connector:          "",
		Enabled:            p.Enabled,
		DocumentURI:        p.DocumentURI,
		Severity:           p.Severity,
		ManualVerification: p.ManualVerification,
		Managed:            p.Managed,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}

	if p.QueryID != nil {
		pa.Query = &api.Query{
			ID: *p.QueryID,
		}
	}

	if v, ok := p.GetTagsMap()[model.KaytuPrivateTagPrefix+"explanation"]; ok && len(v) > 0 {
		pa.Explanation = v[0]
	}
	if v, ok := p.GetTagsMap()[model.KaytuPrivateTagPrefix+"noncompliance-cost"]; ok && len(v) > 0 {
		pa.NonComplianceCost = v[0]
	}
	if v, ok := p.GetTagsMap()[model.KaytuPrivateTagPrefix+"usefulness-example"]; ok && len(v) > 0 {
		pa.UsefulExample = v[0]
	}
	if v, ok := p.GetTagsMap()[model.KaytuPrivateTagPrefix+"manual-remediation"]; ok && len(v) > 0 {
		pa.ManualRemediation = v[0]
	}
	if v, ok := p.GetTagsMap()[model.KaytuPrivateTagPrefix+"cli-remediation"]; ok && len(v) > 0 {
		pa.CliRemediation = v[0]
	}

	return pa
}

func (p Control) GetTagsMap() map[string][]string {
	if p.tagsMap == nil {
		tagLikeArr := make([]model.TagLike, 0, len(p.Tags))
		for _, tag := range p.Tags {
			tagLikeArr = append(tagLikeArr, tag)
		}
		p.tagsMap = model.GetTagsMap(tagLikeArr)
	}
	return p.tagsMap
}

func (p *Control) PopulateConnector(ctx context.Context, db Database, api *api.Control) error {
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
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("control id", p.ID),
	))
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

type ControlTag struct {
	model.Tag
	ControlID string `gorm:"primaryKey"`
}

type BenchmarkControls struct {
	BenchmarkID string
	ControlID   string
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
	PrimaryTable   *string
	ListOfTables   pq.StringArray `gorm:"type:text[]"`
	Engine         string
	Controls       []Control `gorm:"foreignKey:QueryID"`
	Insights       []Insight `gorm:"foreignKey:QueryID"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (q Query) ToApi() api.Query {
	return api.Query{
		ID:             q.ID,
		QueryToExecute: q.QueryToExecute,
		Connector:      source.Type(q.Connector),
		ListOfTables:   q.ListOfTables,
		PrimaryTable:   q.PrimaryTable,
		Engine:         q.Engine,
		CreatedAt:      q.CreatedAt,
		UpdatedAt:      q.UpdatedAt,
	}
}
