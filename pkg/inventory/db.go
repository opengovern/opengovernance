package inventory

import (
	"github.com/jackc/pgx/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	"gorm.io/gorm"
)

type Database struct {
	orm *gorm.DB
}

func (db Database) Initialize() error {
	err := db.orm.AutoMigrate(
		&SmartQuery{},
		&Policy{},
		&PolicyTag{},
		&Benchmark{},
		&BenchmarkTag{},
	)
	if err != nil {
		return err
	}

	return nil
}

// AddQuery adding a query
func (db Database) AddQuery(q *SmartQuery) error {
	tx := db.orm.
		Model(&SmartQuery{}).
		Create(q)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// GetQueries gets list of all queries
func (db Database) GetQueries() ([]SmartQuery, error) {
	var s []SmartQuery
	tx := db.orm.Preload("Tags").Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// GetQueriesWithFilters gets list of all queries filtered by tags and search
func (db Database) GetQueriesWithFilters(search *string, labels []string, provider *api.SourceType) ([]SmartQuery, error) {
	var s []SmartQuery

	m := db.orm.Model(&SmartQuery{}).
		Preload("Tags").
		Joins("LEFT JOIN smartquery_tags on smart_queries.id = smart_query_id " +
			"LEFT JOIN tags on smartquery_tags.tag_id = tags.id ")

	if len(labels) != 0 {
		m = m.Where("tags.value in ?", labels)
	}
	if search != nil {
		m = m.Where("title like ?", "%"+*search+"%")
	}
	if provider != nil {
		m = m.Where("provider = ?", string(*provider))
	}
	tx := m.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	v := map[uint]SmartQuery{}
	for _, item := range s {
		if c, ok := v[item.ID]; ok {
			c.Tags = append(c.Tags, item.Tags...)
		} else {
			v[item.ID] = item
		}
	}
	var res []SmartQuery
	for _, val := range v {
		res = append(res, val)
	}
	return res, nil
}

// CountQueriesWithFilters count list of all queries filtered by tags and search
func (db Database) CountQueriesWithFilters(search *string, labels []string, provider *api.SourceType) (*int64, error) {
	var s int64

	m := db.orm.Model(&SmartQuery{}).
		Preload("Tags").
		Joins("LEFT JOIN smartquery_tags on smart_queries.id = smart_query_id " +
			"LEFT JOIN tags on smartquery_tags.tag_id = tags.id ").
		Distinct("smart_queries.id")

	if len(labels) != 0 {
		m = m.Where("tags.value in ?", labels)
	}
	if search != nil {
		m = m.Where("title like ?", "%"+*search+"%")
	}
	if provider != nil {
		m = m.Where("provider = ?", string(*provider))
	}
	tx := m.Count(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}
	return &s, nil
}

// GetQuery gets a query with matching id
func (db Database) GetQuery(id string) (SmartQuery, error) {
	var s SmartQuery
	tx := db.orm.First(&s, "id = ?", id)

	if tx.Error != nil {
		return SmartQuery{}, tx.Error
	} else if tx.RowsAffected != 1 {
		return SmartQuery{}, pgx.ErrNoRows
	}

	return s, nil
}

// =========== Benchmarks ===========

func (db Database) AddBenchmark(q *Benchmark) error {
	tx := db.orm.Create(q)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) ListBenchmarksWithFilters(provider *string, tags map[string]string) ([]Benchmark, error) {
	var s []Benchmark
	m := db.orm.Model(&Benchmark{}).
		Preload("Tags").
		Preload("Policies")
	if provider != nil {
		m = m.Where("provider = ?", *provider)
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

func (db Database) CountBenchmarksWithFilters(provider *string, tags map[string]string) (int64, error) {
	var s int64
	m := db.orm.Model(&Benchmark{}).
		Preload("Tags").
		Preload("Policies")
	if provider != nil {
		m = m.Where("provider = ?", *provider)
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
	tx := db.orm.Model(&Policy{}).Where("provider = ?", provider).Count(&s)
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
		Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &s, nil
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
