package db

import (
	"fmt"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/types"

	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"

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

		api.Connectors = append(api.Connectors, ca.Connectors...)
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

		api.Connectors = append(api.Connectors, ty)
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

type InsightPeerGroup struct {
	gorm.Model
	Category    string
	Insights    []Insight `gorm:"foreignKey:PeerGroupId;constraint:OnDelete:SET NULL;"`
	ShortTitle  string
	LongTitle   string
	Description string
	LogoURL     *string
	Tags        []InsightTag  `gorm:"many2many:insight_peer_group_tag_rels;"`
	Links       []InsightLink `gorm:"many2many:insight_peer_group_link_rels;"`
}

func (i InsightPeerGroup) ToApi() api.InsightPeerGroup {
	ipg := api.InsightPeerGroup{
		ID:          i.ID,
		Category:    i.Category,
		ShortTitle:  i.ShortTitle,
		LongTitle:   i.LongTitle,
		Description: i.Description,
		LogoURL:     i.LogoURL,
	}

	ipg.Insights = make([]api.Insight, 0, len(i.Insights))
	for _, insight := range i.Insights {
		ipg.Insights = append(ipg.Insights, insight.ToApi())
	}

	ipg.Tags = make([]api.InsightTag, 0, len(i.Tags))
	for _, tag := range i.Tags {
		ipg.Tags = append(ipg.Tags, api.InsightTag{
			ID:    tag.ID,
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	ipg.Links = make([]api.InsightLink, 0, len(i.Links))
	for _, link := range i.Links {
		ipg.Links = append(ipg.Links, api.InsightLink{
			ID:   link.ID,
			Text: link.Text,
			URI:  link.URI,
		})
	}
	return ipg
}

type InsightPeerGroupTagRel struct {
	InsightPeerGroupID uint
	InsightTagID       uint
}

type InsightPeerGroupLinkRel struct {
	InsightPeerGroupID uint
	InsightLinkID      uint
}

type Insight struct {
	gorm.Model
	PeerGroupId *uint
	QueryID     string
	Query       Query `gorm:"foreignKey:QueryID;references:ID;constraint:OnDelete:CASCADE;"`
	Category    string
	Connector   source.Type
	ShortTitle  string
	LongTitle   string
	Description string
	LogoURL     *string
	Tags        []InsightTag  `gorm:"many2many:insight_tag_rels;"`
	Links       []InsightLink `gorm:"many2many:insight_link_rels;"`
	Enabled     bool          `gorm:"default:true"`
	Internal    bool
}

func (i Insight) ToApi() api.Insight {
	ia := api.Insight{
		ID:          i.ID,
		PeerGroupId: i.PeerGroupId,
		Query:       i.Query.ToApi(),
		Category:    i.Category,
		Connector:   i.Connector,
		ShortTitle:  i.ShortTitle,
		LongTitle:   i.LongTitle,
		Description: i.Description,
		LogoURL:     i.LogoURL,
		Tags:        nil,
		Links:       nil,
		Enabled:     i.Enabled,
		Internal:    i.Internal,
	}

	ia.Tags = make([]api.InsightTag, 0, len(i.Tags))
	for _, tag := range i.Tags {
		ia.Tags = append(ia.Tags, api.InsightTag{
			ID:    tag.ID,
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	ia.Links = make([]api.InsightLink, 0, len(i.Links))
	for _, link := range i.Links {
		ia.Links = append(ia.Links, api.InsightLink{
			ID:   link.ID,
			Text: link.Text,
			URI:  link.URI,
		})
	}
	return ia
}

type InsightTagRel struct {
	InsightID    uint
	InsightTagID uint
}

type InsightLinkRel struct {
	InsightID     uint
	InsightLinkID uint
}

type InsightTag struct {
	gorm.Model
	Key               string
	Value             string
	Insights          []Insight          `gorm:"many2many:insight_tag_rels;"`
	InsightPeerGroups []InsightPeerGroup `gorm:"many2many:insight_peer_group_tag_rels;"`
}

type InsightLink struct {
	gorm.Model
	Insights          []Insight          `gorm:"many2many:insight_link_rels;"`
	InsightPeerGroups []InsightPeerGroup `gorm:"many2many:insight_peer_group_link_rels;"`
	Text              string
	URI               string
}

type Query struct {
	ID             string `gorm:"primarykey"`
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
