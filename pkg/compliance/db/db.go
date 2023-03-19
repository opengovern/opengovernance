package db

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Database struct {
	Orm *gorm.DB
}

func (db Database) Initialize() error {
	err := db.Orm.AutoMigrate(
		&Query{},
		&PolicyTag{},
		&BenchmarkTag{},
		&Policy{},
		&Benchmark{},
		&BenchmarkAssignment{},
	)
	if err != nil {
		return err
	}

	return nil
}

// =========== Benchmarks ===========

func (db Database) ListBenchmarks() ([]Benchmark, error) {
	var s []Benchmark
	tx := db.Orm.Model(&Benchmark{}).
		Preload("Tags").
		Preload("Children").
		Preload("Policies").
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) GetBenchmark(benchmarkId string) (*Benchmark, error) {
	var s Benchmark
	tx := db.Orm.Model(&Benchmark{}).
		Preload("Tags").
		Preload("Children").
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
	tx := db.Orm.Model(&Benchmark{}).
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
	tx := db.Orm.Model(&Policy{}).
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

func (db Database) GetPolicy(id string) (*Policy, error) {
	var s Policy
	tx := db.Orm.Model(&Policy{}).
		Preload("Tags").
		Where("id = ?", id).
		First(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &s, nil
}

func (db Database) ListPoliciesByBenchmarkID(benchmarkID string) ([]Policy, error) {
	var s []Policy
	tx := db.Orm.Model(&Policy{}).
		Preload("Tags").
		Preload("Benchmarks").
		Where(Policy{Benchmarks: []Benchmark{{ID: benchmarkID}}}).Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

// =========== BenchmarkAssignment ===========

func (db Database) AddBenchmarkAssignment(assignment *BenchmarkAssignment) error {
	tx := db.Orm.Where(BenchmarkAssignment{
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
	tx := db.Orm.Model(&BenchmarkAssignment{}).Where(BenchmarkAssignment{SourceId: sourceId}).Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) GetBenchmarkAssignmentsByBenchmarkId(benchmarkId string) ([]BenchmarkAssignment, error) {
	var s []BenchmarkAssignment
	tx := db.Orm.Model(&BenchmarkAssignment{}).Where(BenchmarkAssignment{BenchmarkId: benchmarkId}).Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) ListBenchmarkAssignments() ([]BenchmarkAssignment, error) {
	var s []BenchmarkAssignment
	tx := db.Orm.Model(&BenchmarkAssignment{}).Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) GetBenchmarkAssignmentByIds(sourceId uuid.UUID, benchmarkId string) (*BenchmarkAssignment, error) {
	var s BenchmarkAssignment
	tx := db.Orm.Model(&BenchmarkAssignment{}).Where(BenchmarkAssignment{BenchmarkId: benchmarkId, SourceId: sourceId}).Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &s, nil
}

func (db Database) DeleteBenchmarkAssignmentById(sourceId uuid.UUID, benchmarkId string) error {
	tx := db.Orm.Unscoped().Where(BenchmarkAssignment{BenchmarkId: benchmarkId, SourceId: sourceId}).Delete(&BenchmarkAssignment{})

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
