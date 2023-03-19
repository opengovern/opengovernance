package db

import (
	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	"time"

	"gorm.io/gorm"
)

type BenchmarkAssignment struct {
	gorm.Model
	BenchmarkId string    `gorm:"index:idx_benchmark_source"`
	SourceId    uuid.UUID `gorm:"index:idx_benchmark_source"`
	AssignedAt  time.Time
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
	return ba
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
	Severity           string
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

type Query struct {
	ID             string `gorm:"primarykey"`
	QueryToExecute string
	Connector      string
	ListOfTables   string
	Engine         string
	Policies       []Policy `gorm:"foreignKey:QueryID"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
