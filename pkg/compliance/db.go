package compliance

import (
	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gorm.io/gorm"
)

type Database struct {
	orm *gorm.DB
}

func (db Database) Initialize() error {
	err := db.orm.AutoMigrate(
		&Policy{},
		&PolicyTag{},
		&Benchmark{},
		&BenchmarkTag{},
		&BenchmarkAssignment{},
	)
	if err != nil {
		return err
	}

	return nil
}

// =========== Benchmarks ===========

func (db Database) AddBenchmark(q *Benchmark) error {
	tx := db.orm.Create(q)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) ListBenchmarksWithFilters(provider source.Type, tags map[string]string) ([]Benchmark, error) {
	var s []Benchmark
	m := db.orm.Model(&Benchmark{}).
		Preload("Tags").
		Preload("Policies")
	if !provider.IsNull() {
		m = m.Where("provider = ?", provider.String())
	}
	for key, value := range tags {
		m = m.Where("id IN (SELECT benchmark_id FROM benchmark_tag_rel WHERE benchmark_tag_id IN (SELECT id FROM benchmark_tags WHERE key = ? AND value = ?))", key, value)
	}

	tx := m.Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) ListBenchmarks() ([]Benchmark, error) {
	var s []Benchmark
	tx := db.orm.Model(&Benchmark{}).Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) CountBenchmarksWithFilters(provider source.Type, tags map[string]string) (int64, error) {
	var s int64
	m := db.orm.Model(&Benchmark{}).
		Preload("Tags").
		Preload("Policies")
	if !provider.IsNull() {
		m = m.Where("provider = ?", provider.String())
	}
	for key, value := range tags {
		m = m.Where("id IN (SELECT benchmark_id FROM benchmark_tag_rel WHERE benchmark_tag_id IN (SELECT id FROM benchmark_tags WHERE key = ? AND value = ?))", key, value)
	}

	tx := m.Count(&s)
	if tx.Error != nil {
		return 0, tx.Error
	}

	return s, nil
}

func (db Database) CountPolicies(provider string) (int64, error) {
	var s int64
	tx := db.orm.Model(&Policy{})
	if provider != "" {
		tx = tx.Where("provider = ?", provider)
	}
	tx = tx.Count(&s)

	if tx.Error != nil {
		return 0, tx.Error
	}
	return s, nil
}

func (db Database) GetBenchmark(benchmarkId string) (*Benchmark, error) {
	var s Benchmark
	tx := db.orm.Model(&Benchmark{}).
		Preload("Tags").
		Preload("Policies").
		Where("id = ?", benchmarkId).
		First(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &s, nil
}

func (db Database) GetBenchmarksTitle(ds []string) (map[string]string, error) {
	var bs []Benchmark
	tx := db.orm.Model(&Benchmark{}).
		Where("id in ?", ds).
		Select("id, title").
		Find(&bs)

	if tx.Error != nil {
		return nil, tx.Error
	}

	res := map[string]string{}
	for _, b := range bs {
		res[b.ID] = b.Title
	}
	return res, nil
}

func (db Database) GetPoliciesTitle(ds []string) (map[string]string, error) {
	var bs []Policy
	tx := db.orm.Model(&Policy{}).
		Where("id in ?", ds).
		Select("id, title").
		Find(&bs)

	if tx.Error != nil {
		return nil, tx.Error
	}

	res := map[string]string{}
	for _, b := range bs {
		res[b.ID] = b.Title
	}
	return res, nil
}

// =========== Policy ===========

func (db Database) GetPoliciesWithFilters(benchmarkId string,
	category, subcategory, section, severity *string) ([]Policy, error) {
	var s []Policy
	m := db.orm.Model(&Policy{}).
		Preload("Tags").
		Where("id IN (SELECT policy_id FROM benchmark_policies WHERE benchmark_id = ?)", benchmarkId)

	if category != nil {
		m = m.Where("category = ?", *category)
	}
	if subcategory != nil {
		m = m.Where("subcategory = ?", *subcategory)
	}
	if section != nil {
		m = m.Where("section = ?", *section)
	}
	if severity != nil {
		m = m.Where("severity = ?", *severity)
	}
	tx := m.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// =========== Benchmark Tags ===========

func (db Database) ListBenchmarkTags() ([]BenchmarkTag, error) {
	var s []BenchmarkTag
	tx := db.orm.
		Preload("Benchmarks").
		Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// =========== BenchmarkAssignment ===========

func (db Database) AddBenchmarkAssignment(assignment *BenchmarkAssignment) error {
	tx := db.orm.Where(BenchmarkAssignment{
		BenchmarkId: assignment.BenchmarkId,
		SourceId:    assignment.SourceId,
	}).FirstOrCreate(assignment)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) GetBenchmarkAssignmentsBySourceId(sourceId uuid.UUID) ([]BenchmarkAssignment, error) {
	var s []BenchmarkAssignment
	tx := db.orm.Model(&BenchmarkAssignment{}).Where(BenchmarkAssignment{SourceId: sourceId}).Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) GetBenchmarkAssignmentsByBenchmarkId(benchmarkId string) ([]BenchmarkAssignment, error) {
	var s []BenchmarkAssignment
	tx := db.orm.Model(&BenchmarkAssignment{}).Where(BenchmarkAssignment{BenchmarkId: benchmarkId}).Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) GetBenchmarkAssignmentByIds(sourceId uuid.UUID, benchmarkId string) (*BenchmarkAssignment, error) {
	var s BenchmarkAssignment
	tx := db.orm.Model(&BenchmarkAssignment{}).Where(BenchmarkAssignment{BenchmarkId: benchmarkId, SourceId: sourceId}).Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &s, nil
}

func (db Database) DeleteBenchmarkAssignmentById(sourceId uuid.UUID, benchmarkId string) error {
	tx := db.orm.Unscoped().Where(BenchmarkAssignment{BenchmarkId: benchmarkId, SourceId: sourceId}).Delete(&BenchmarkAssignment{})

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
