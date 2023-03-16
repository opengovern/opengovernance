package db

import (
	"github.com/google/uuid"
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
	Tags        []BenchmarkTag `gorm:"many2many:benchmark_tag_rel;"`
	Children    []Benchmark    `gorm:"many2many:benchmark_children;"`
	Policies    []Policy       `gorm:"many2many:benchmark_policies;"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type BenchmarkChild struct {
	BenchmarkID string
	ChildID     string
}

type BenchmarkTag struct {
	gorm.Model
	Key        string
	Value      string
	Benchmarks []Benchmark `gorm:"many2many:benchmark_tag_rel;"`
}

type BenchmarkTagRel struct {
	BenchmarkID    string
	BenchmarkTagID uint
}

type Policy struct {
	ID                 string `gorm:"primarykey"`
	Title              string
	Description        string
	Tags               []PolicyTag `gorm:"many2many:policy_tag_rel;"`
	DocumentURI        string
	QueryID            *string
	Benchmarks         []Benchmark `gorm:"many2many:benchmark_policies;"`
	Severity           string
	ManualVerification bool
	Managed            bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type PolicyTag struct {
	gorm.Model
	Key   string
	Value string
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
	EngineVersion  string
	Policies       []Policy `gorm:"foreignKey:QueryID"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
