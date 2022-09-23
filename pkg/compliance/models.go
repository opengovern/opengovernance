package compliance

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Benchmark struct {
	gorm.Model
	ID          string
	Title       string
	Description string
	Provider    string
	Enabled     bool
	Tags        []BenchmarkTag `gorm:"many2many:benchmark_tag_rel;"`
	Policies    []Policy       `gorm:"many2many:benchmark_policies;"`
}

type BenchmarkTag struct {
	gorm.Model
	Key        string
	Value      string
	Benchmarks []Benchmark `gorm:"many2many:benchmark_tag_rel;"`
}

type BenchmarkAssignment struct {
	gorm.Model
	BenchmarkId string    `gorm:"index:idx_benchmark_source"`
	SourceId    uuid.UUID `gorm:"index:idx_benchmark_source"`
	AssignedAt  time.Time
}

type Policy struct {
	gorm.Model
	ID                    string
	Title                 string
	Description           string
	Tags                  []PolicyTag `gorm:"many2many:policy_tag_rel;"`
	Provider              string
	Category              string
	SubCategory           string
	Section               string
	Severity              string
	ManualVerification    string
	ManualRemedation      string
	CommandLineRemedation string
	QueryToRun            string
	KeibiManaged          bool
	Benchmarks            []Benchmark `gorm:"many2many:benchmark_policies;"`
}

type PolicyTag struct {
	gorm.Model
	Key   string
	Value string
}
