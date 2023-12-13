package db

import (
	"errors"

	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Database struct {
	Orm *gorm.DB
}

func (db Database) Initialize() error {
	err := db.Orm.AutoMigrate(
		&Query{},
		&Control{},
		&ControlTag{},
		&Benchmark{},
		&BenchmarkTag{},
		&Insight{},
		&InsightTag{},
		&InsightGroup{},
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
	tx := db.Orm.Model(&Benchmark{}).Preload(clause.Associations).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) ListBenchmarksBare() ([]Benchmark, error) {
	var s []Benchmark
	tx := db.Orm.Model(&Benchmark{}).Preload("Tags").
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

// ListRootBenchmarks returns all benchmarks that are not children of any other benchmark
// is it important to note that this function does not return the children of the root benchmarks neither the controls
func (db Database) ListRootBenchmarks(tags map[string][]string) ([]Benchmark, error) {
	var benchmarks []Benchmark
	tx := db.Orm.Model(&Benchmark{}).Preload(clause.Associations).
		Where("NOT EXISTS (SELECT 1 FROM benchmark_children WHERE benchmark_children.child_id = benchmarks.id)")
	if len(tags) > 0 {
		tx = tx.Joins("JOIN benchmark_tags AS tags ON tags.benchmark_id = benchmarks.id")
		for key, values := range tags {
			if len(values) != 0 {
				tx = tx.Where("tags.key = ? AND (tags.value && ?)", key, pq.StringArray(values))
			} else {
				tx = tx.Where("tags.key = ?", key)
			}
		}
	}
	err := tx.Find(&benchmarks).Error
	if err != nil {
		return nil, err
	}

	return benchmarks, nil
}

func (db Database) GetBenchmark(benchmarkId string) (*Benchmark, error) {
	var s Benchmark
	tx := db.Orm.Model(&Benchmark{}).Preload(clause.Associations).
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

func (db Database) GetBenchmarkBare(benchmarkId string) (*Benchmark, error) {
	var s Benchmark
	tx := db.Orm.Model(&Benchmark{}).Preload("Tags").
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

func (db Database) GetControlsTitle(ds []string) (map[string]string, error) {
	var bs []Control
	tx := db.Orm.Model(&Control{}).
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

// =========== Control ===========

func (db Database) GetControl(id string) (*Control, error) {
	var s Control
	tx := db.Orm.Model(&Control{}).
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

func (db Database) ListControlsByBenchmarkID(benchmarkID string) ([]Control, error) {
	var s []Control
	tx := db.Orm.Model(&Control{}).
		Preload("Tags").
		Preload("Benchmarks").
		Where(Control{Benchmarks: []Benchmark{{ID: benchmarkID}}}).Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) GetControls(controlIDs []string) ([]Control, error) {
	var s []Control
	tx := db.Orm.Model(&Control{}).
		Preload("Tags").
		Where("id IN ?", controlIDs).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) GetQueries(queryIDs []string) ([]Query, error) {
	var s []Query
	tx := db.Orm.Model(&Query{}).
		Where("id IN ?", queryIDs).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) GetQueriesIdAndConnector(queryIDs []string) ([]Query, error) {
	var s []Query
	tx := db.Orm.Model(&Query{}).
		Select("id, connector").
		Where("id IN ?", queryIDs).
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

func (db Database) GetBenchmarkAssignmentsByConnectionId(connectionId string) ([]BenchmarkAssignment, error) {
	var s []BenchmarkAssignment
	tx := db.Orm.Model(&BenchmarkAssignment{}).
		Where(BenchmarkAssignment{ConnectionId: &connectionId}).
		Where("resource_collection IS NULL").Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) GetBenchmarkAssignmentsByResourceCollectionId(resourceCollectionId string) ([]BenchmarkAssignment, error) {
	var s []BenchmarkAssignment
	tx := db.Orm.Model(&BenchmarkAssignment{}).
		Where(BenchmarkAssignment{ResourceCollection: &resourceCollectionId}).
		Where("connection_id IS NULL").Scan(&s)

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

func (db Database) GetBenchmarkAssignmentByIds(benchmarkId string, connectionId, resourceCollectionId *string) (*BenchmarkAssignment, error) {
	var s BenchmarkAssignment
	tx := db.Orm.Model(&BenchmarkAssignment{}).Where(BenchmarkAssignment{
		BenchmarkId:        benchmarkId,
		ConnectionId:       connectionId,
		ResourceCollection: resourceCollectionId,
	}).Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &s, nil
}

func (db Database) DeleteBenchmarkAssignmentByIds(benchmarkId string, connectionId, resourceCollectionId *string) error {
	tx := db.Orm.Unscoped().Where(BenchmarkAssignment{
		BenchmarkId:        benchmarkId,
		ConnectionId:       connectionId,
		ResourceCollection: resourceCollectionId,
	}).Delete(&BenchmarkAssignment{})

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) DeleteBenchmarkAssignmentByBenchmarkId(benchmarkId string) error {
	tx := db.Orm.Unscoped().Where(BenchmarkAssignment{BenchmarkId: benchmarkId}).Delete(&BenchmarkAssignment{})

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) ListComplianceTagKeysWithPossibleValues() (map[string][]string, error) {
	var tags []BenchmarkTag
	tx := db.Orm.Model(BenchmarkTag{}).Find(&tags)
	if tx.Error != nil {
		return nil, tx.Error
	}
	tagLikes := make([]model.TagLike, 0, len(tags))
	for _, tag := range tags {
		tagLikes = append(tagLikes, tag)
	}
	result := model.GetTagsMap(tagLikes)
	return result, nil
}

func (db Database) ListInsightTagKeysWithPossibleValues() (map[string][]string, error) {
	var tags []InsightTag
	tx := db.Orm.Model(InsightTag{}).Find(&tags)
	if tx.Error != nil {
		return nil, tx.Error
	}
	tagLikes := make([]model.TagLike, 0, len(tags))
	for _, tag := range tags {
		tagLikes = append(tagLikes, tag)
	}
	result := model.GetTagsMap(tagLikes)
	return result, nil
}

func (db Database) GetInsightTagTagPossibleValues(key string) ([]string, error) {
	var tags []InsightTag
	tx := db.Orm.Model(InsightTag{}).Where("key = ?", key).Find(&tags)
	if tx.Error != nil {
		return nil, tx.Error
	}
	tagLikes := make([]model.TagLike, 0, len(tags))
	for _, tag := range tags {
		tagLikes = append(tagLikes, tag)
	}
	result := model.GetTagsMap(tagLikes)
	return result[key], nil
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

func (db Database) ListInsightsWithFilters(insightIDs []uint, connectors []source.Type, enabled *bool, tags map[string][]string) ([]Insight, error) {
	var s []Insight
	m := db.Orm.Model(&Insight{}).Preload(clause.Associations)
	if len(insightIDs) > 0 {
		m = m.Where("id IN ?", insightIDs)
	}
	if len(connectors) > 0 {
		m = m.Where("connector IN ?", connectors)
	}
	if enabled != nil {
		m = m.Where("enabled = ?", *enabled)
	}
	if len(tags) > 0 {
		m = m.Joins("JOIN insight_tags AS tags ON tags.insight_id = insights.id")
		for key, values := range tags {
			if len(values) != 0 {
				m = m.Where("tags.key = ? AND (tags.value && ?)", key, pq.StringArray(values))
			} else {
				m = m.Where("tags.key = ?", key)
			}
		}
	}
	tx := m.Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) ListInsightGroups(connectors []source.Type, tags map[string][]string) ([]InsightGroup, error) {
	var insightGroups []InsightGroup
	m := db.Orm.Model(&InsightGroup{}).Preload(clause.Associations).Find(&insightGroups)
	if m.Error != nil {
		return nil, m.Error
	}

	insightIDs := make([]uint, 0)
	for _, insightGroup := range insightGroups {
		for _, insight := range insightGroup.Insights {
			insightIDs = append(insightIDs, insight.ID)
		}
	}

	var insights []Insight
	insights, err := db.ListInsightsWithFilters(insightIDs, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	insightMap := make(map[uint]Insight)
	for i, insight := range insights {
		insightMap[insight.ID] = insights[i]
	}

	for i, insightGroup := range insightGroups {
		for j, insight := range insightGroup.Insights {
			insightGroup.Insights[j] = insightMap[insight.ID]
		}
		insightGroups[i] = insightGroup
	}

	filteredInsightGroups := make([]InsightGroup, 0)
	for _, insightGroup := range insightGroups {
		insightGroupTags := make([]model.TagLike, 0)
		insightGroupConnectorMap := make(map[source.Type]bool)
		for _, insight := range insightGroup.Insights {
			insightGroupConnectorMap[insight.Connector] = true
			for _, tag := range insight.Tags {
				insightGroupTags = append(insightGroupTags, tag)
			}
		}

		doAdd := true
		for _, connector := range connectors {
			if _, ok := insightGroupConnectorMap[connector]; !ok {
				doAdd = false
				break
			}
		}
		if !doAdd {
			continue
		}
		insightGroupTagsMap := model.GetTagsMap(insightGroupTags)
		for key, values := range tags {
			v, ok := insightGroupTagsMap[key]
			if !ok {
				doAdd = false
				break
			}
			if len(values) == 0 {
				continue
			}
			for _, value := range values {
				found := false
				for _, vv := range v {
					if vv == value {
						found = true
						break
					}
				}
				if !found {
					doAdd = false
					break
				}
			}
			if !doAdd {
				break
			}
		}
		if doAdd {
			filteredInsightGroups = append(filteredInsightGroups, insightGroup)
		}
	}
	insightGroups = filteredInsightGroups

	return insightGroups, nil
}

func (db Database) GetInsightGroup(id uint) (*InsightGroup, error) {
	var res InsightGroup
	tx := db.Orm.Model(&InsightGroup{}).Preload(clause.Associations).
		Where("id = ?", id).
		First(&res)
	if tx.Error != nil {
		return nil, tx.Error
	}
	insightIDs := make([]uint, 0, len(res.Insights))
	for _, insight := range res.Insights {
		insightIDs = append(insightIDs, insight.ID)
	}
	insights, err := db.ListInsightsWithFilters(insightIDs, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	insightMap := make(map[uint]Insight)
	for i, insight := range insights {
		insightMap[insight.ID] = insights[i]
	}

	res.Insights = make([]Insight, 0, len(insightMap))
	for _, insight := range insightMap {
		res.Insights = append(res.Insights, insight)
	}

	return &res, nil
}

func (db Database) ListControls() ([]Control, error) {
	var s []Control
	tx := db.Orm.Model(&Control{}).Preload(clause.Associations).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) ListQueries() ([]Query, error) {
	var s []Query
	tx := db.Orm.Model(&Query{}).Preload(clause.Associations).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) ListControlsBare() ([]Control, error) {
	var s []Control
	tx := db.Orm.Model(&Control{}).Preload("Tags").
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}
