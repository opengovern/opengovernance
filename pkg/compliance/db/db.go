package db

import (
	"errors"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Database struct {
	Orm *gorm.DB
}

func (db Database) Initialize() error {
	err := db.Orm.AutoMigrate(
		&Query{},
		&PolicyTag{},
		&BenchmarkTag{},
		&InsightTag{},
		&InsightLink{},
		&InsightPeerGroup{},
		&Insight{},
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

func (db Database) ListRootBenchmarks() ([]Benchmark, error) {
	benchmarks, err := db.ListBenchmarks()
	if err != nil {
		return nil, err
	}

	var response []Benchmark
	for _, b := range benchmarks {
		hasParent := false
		for _, parent := range benchmarks {
			for _, child := range parent.Children {
				if child.ID == b.ID {
					hasParent = true
				}
			}
		}

		if !hasParent {
			response = append(response, b)
		}
	}

	return response, nil
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
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}

	return &s, nil
}

func (db Database) GetQuery(queryID string) (*Query, error) {
	var s Query
	tx := db.Orm.Model(&Query{}).
		Where("id = ?", queryID).
		First(&s)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
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
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
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

func (db Database) GetPolicies(policyIDs []string) ([]Policy, error) {
	var s []Policy
	tx := db.Orm.Model(&Policy{}).
		Preload("Tags").
		Where("id IN ?", policyIDs).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

// =========== BenchmarkAssignment ===========

func (db Database) AddBenchmarkAssignment(assignment *BenchmarkAssignment) error {
	tx := db.Orm.Where(BenchmarkAssignment{
		BenchmarkId:  assignment.BenchmarkId,
		ConnectionId: assignment.ConnectionId,
	}).FirstOrCreate(assignment)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) GetBenchmarkAssignmentsBySourceId(connectionId string) ([]BenchmarkAssignment, error) {
	var s []BenchmarkAssignment
	tx := db.Orm.Model(&BenchmarkAssignment{}).Where(BenchmarkAssignment{ConnectionId: connectionId}).Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) GetBenchmarkAssignmentsByBenchmarkId(benchmarkId string) ([]BenchmarkAssignment, error) {
	var s []BenchmarkAssignment
	tx := db.Orm.Model(&BenchmarkAssignment{}).Where(BenchmarkAssignment{BenchmarkId: benchmarkId}).Scan(&s)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
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

func (db Database) GetBenchmarkAssignmentByIds(connectionId string, benchmarkId string) (*BenchmarkAssignment, error) {
	var s BenchmarkAssignment
	tx := db.Orm.Model(&BenchmarkAssignment{}).Where(BenchmarkAssignment{BenchmarkId: benchmarkId, ConnectionId: connectionId}).Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &s, nil
}

func (db Database) DeleteBenchmarkAssignmentById(connectionId string, benchmarkId string) error {
	tx := db.Orm.Unscoped().Where(BenchmarkAssignment{BenchmarkId: benchmarkId, ConnectionId: connectionId}).Delete(&BenchmarkAssignment{})

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) GetInsight(id uint) (*Insight, error) {
	var res Insight
	tx := db.Orm.Model(&Insight{}).Preload(clause.Associations).
		Where("id = ?", id).
		First(&res)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &res, nil
}

func (db Database) ListInsightsWithFilters(connector source.Type, enabled *bool) ([]Insight, error) {
	var s []Insight
	m := db.Orm.Model(&Insight{}).Preload(clause.Associations)
	if connector != source.Nil {
		m = m.Where("connector = ?", connector)
	}
	if enabled != nil {
		m = m.Where("enabled = ?", *enabled)
	}
	tx := m.Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) ListInsightsPeerGroups() ([]InsightPeerGroup, error) {
	var s []InsightPeerGroup
	tx := db.Orm.Model(&InsightPeerGroup{}).Preload(clause.Associations).Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) GetInsightsPeerGroup(id uint) (*InsightPeerGroup, error) {
	var res InsightPeerGroup
	tx := db.Orm.Model(&InsightPeerGroup{}).Preload(clause.Associations).
		Where("id = ?", id).
		First(&res)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &res, nil
}
