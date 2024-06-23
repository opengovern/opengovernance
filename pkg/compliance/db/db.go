package db

import (
	"context"
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

func (db Database) Initialize(ctx context.Context) error {
	err := db.Orm.WithContext(ctx).AutoMigrate(
		&Query{},
		&QueryParameter{},
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

func (db Database) ListBenchmarks(ctx context.Context) ([]Benchmark, error) {
	var s []Benchmark
	tx := db.Orm.WithContext(ctx).Model(&Benchmark{}).Preload(clause.Associations).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) ListBenchmarksBare(ctx context.Context) ([]Benchmark, error) {
	var s []Benchmark
	tx := db.Orm.WithContext(ctx).Model(&Benchmark{}).Preload("Tags").
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

// ListRootBenchmarks returns all benchmarks that are not children of any other benchmark
// is it important to note that this function does not return the children of the root benchmarks neither the controls
func (db Database) ListRootBenchmarks(ctx context.Context, tags map[string][]string) ([]Benchmark, error) {
	var benchmarks []Benchmark
	tx := db.Orm.WithContext(ctx).Model(&Benchmark{}).Preload(clause.Associations).
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

func (db Database) ListRootBenchmarksWithSubtreeControls(ctx context.Context, tags map[string][]string) ([]Benchmark, error) {
	var benchmarks []Benchmark

	allBenchmarks, err := db.ListBenchmarks(ctx)
	if err != nil {
		return nil, err
	}
	allBenchmarksMap := make(map[string]Benchmark)
	for _, b := range allBenchmarks {
		allBenchmarksMap[b.ID] = b
	}

	var populateControls func(ctx context.Context, benchmark *Benchmark) error
	populateControls = func(ctx context.Context, benchmark *Benchmark) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if benchmark == nil {
			return nil
		}
		if len(benchmark.Children) > 0 {
			for _, child := range benchmark.Children {
				child := allBenchmarksMap[child.ID]
				err := populateControls(ctx, &child)
				if err != nil {
					return err
				}
				for _, control := range child.Controls {
					found := false
					for _, c := range benchmark.Controls {
						if c.ID == control.ID {
							found = true
							break
						}
					}
					if !found {
						benchmark.Controls = append(benchmark.Controls, control)
					}
				}
			}
		}
		return nil
	}

	rootBenchmarks, err := db.ListRootBenchmarks(ctx, tags)
	if err != nil {
		return nil, err
	}

	for _, b := range rootBenchmarks {
		err := populateControls(ctx, &b)
		if err != nil {
			return nil, err
		}
		benchmarks = append(benchmarks, b)
	}

	return benchmarks, nil
}

func (db Database) GetBenchmark(ctx context.Context, benchmarkId string) (*Benchmark, error) {
	var s Benchmark
	tx := db.Orm.WithContext(ctx).Model(&Benchmark{}).Preload(clause.Associations).
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

func (db Database) GetBenchmarkBare(ctx context.Context, benchmarkId string) (*Benchmark, error) {
	var s Benchmark
	tx := db.Orm.WithContext(ctx).Model(&Benchmark{}).Preload("Tags").
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

func (db Database) GetBenchmarksBare(ctx context.Context, benchmarkIds []string) ([]Benchmark, error) {
	var s []Benchmark
	tx := db.Orm.WithContext(ctx).Model(&Benchmark{}).Preload("Tags").
		Where("id in ?", benchmarkIds).
		Find(&s)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) SetBenchmarkAutoAssign(ctx context.Context, benchmarkId string, autoAssign bool) error {
	tx := db.Orm.WithContext(ctx).Model(&Benchmark{}).Where("id = ?", benchmarkId).Update("auto_assign", autoAssign)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) ListDistinctRootBenchmarksFromControlIds(ctx context.Context, controlIds []string) ([]Benchmark, error) {
	s := make(map[string]Benchmark)

	findControls := make(map[string]struct{})
	for _, controlId := range controlIds {
		findControls[controlId] = struct{}{}
	}

	rootBenchmarksWithControls, err := db.ListRootBenchmarksWithSubtreeControls(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, b := range rootBenchmarksWithControls {
		for _, c := range b.Controls {
			if _, ok := findControls[c.ID]; ok {
				s[b.ID] = b
				break
			}
		}
	}

	var res []Benchmark
	for _, b := range s {
		res = append(res, b)
	}

	return res, nil
}

func (db Database) GetQuery(ctx context.Context, queryID string) (*Query, error) {
	var s Query
	tx := db.Orm.WithContext(ctx).Model(&Query{}).Preload(clause.Associations).
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

func (db Database) GetBenchmarksTitle(ctx context.Context, ds []string) (map[string]string, error) {
	var bs []Benchmark
	tx := db.Orm.WithContext(ctx).Model(&Benchmark{}).
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

func (db Database) GetControlsTitle(ctx context.Context, ds []string) (map[string]string, error) {
	var bs []Control
	tx := db.Orm.WithContext(ctx).Model(&Control{}).
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

func (db Database) GetControl(ctx context.Context, id string) (*Control, error) {
	var s Control
	tx := db.Orm.WithContext(ctx).Model(&Control{}).
		Preload(clause.Associations).
		Where("id = ?", id).
		First(&s)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}

	if s.QueryID != nil {
		var query Query
		tx := db.Orm.WithContext(ctx).Model(&Query{}).Preload(clause.Associations).Where("id = ?", *s.QueryID).First(&query)
		if tx.Error != nil {
			return nil, tx.Error
		}
		s.Query = &query
	}

	return &s, nil
}

func (db Database) ListControlsByBenchmarkID(ctx context.Context, benchmarkID string) ([]Control, error) {
	var s []Control
	tx := db.Orm.WithContext(ctx).Model(&Control{}).
		Preload("Tags").
		Preload("Benchmarks").
		Where(Control{Benchmarks: []Benchmark{{ID: benchmarkID}}}).Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}

	queryIds := make([]string, 0, len(s))
	for _, control := range s {
		if control.QueryID != nil {
			queryIds = append(queryIds, *control.QueryID)
		}
	}
	var queriesMap map[string]Query
	if len(queryIds) > 0 {
		var queries []Query
		qtx := db.Orm.WithContext(ctx).Model(&Query{}).Preload(clause.Associations).Where("id IN ?", queryIds).Find(&queries)
		if qtx.Error != nil {
			return nil, qtx.Error
		}
		queriesMap = make(map[string]Query)
		for _, query := range queries {
			queriesMap[query.ID] = query
		}
	}

	for i, c := range s {
		if c.QueryID != nil {
			v := queriesMap[*c.QueryID]
			s[i].Query = &v
		}
	}

	return s, nil
}

func (db Database) GetControls(ctx context.Context, controlIDs []string, tags map[string][]string) ([]Control, error) {
	var s []Control
	tx := db.Orm.WithContext(ctx).Model(&Control{}).
		Preload(clause.Associations)
	if len(controlIDs) > 0 {
		tx = tx.Where("id IN ?", controlIDs)
	}
	if len(tags) > 0 {
		tx = tx.Joins("JOIN control_tags AS tags ON tags.control_id = controls.id")
		for key, values := range tags {
			if len(values) != 0 {
				tx = tx.Where("tags.key = ? AND (tags.value && ?)", key, pq.StringArray(values))
			} else {
				tx = tx.Where("tags.key = ?", key)
			}
		}
	}
	if tx.Find(&s).Error != nil {
		return nil, tx.Error
	}

	queryIds := make([]string, 0, len(s))
	for _, control := range s {
		if control.QueryID != nil {
			queryIds = append(queryIds, *control.QueryID)
		}
	}
	var queriesMap map[string]Query
	if len(queryIds) > 0 {
		var queries []Query
		qtx := db.Orm.WithContext(ctx).Model(&Query{}).Preload(clause.Associations).Where("id IN ?", queryIds).Find(&queries)
		if qtx.Error != nil {
			return nil, qtx.Error
		}
		queriesMap = make(map[string]Query)
		for _, query := range queries {
			queriesMap[query.ID] = query
		}
	}

	for i, c := range s {
		if c.QueryID != nil {
			v := queriesMap[*c.QueryID]
			s[i].Query = &v
		}
	}

	return s, nil
}

func (db Database) GetQueries(ctx context.Context, queryIDs []string) ([]Query, error) {
	var s []Query
	tx := db.Orm.WithContext(ctx).Model(&Query{}).
		Where("id IN ?", queryIDs).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) GetQueriesIdAndConnector(ctx context.Context, queryIDs []string) ([]Query, error) {
	var s []Query
	tx := db.Orm.WithContext(ctx).Model(&Query{}).
		Select("id, connector").
		Where("id IN ?", queryIDs).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

// =========== BenchmarkAssignment ===========

func (db Database) AddBenchmarkAssignment(ctx context.Context, assignment *BenchmarkAssignment) error {
	tx := db.Orm.WithContext(ctx).Where(BenchmarkAssignment{
		BenchmarkId:  assignment.BenchmarkId,
		ConnectionId: assignment.ConnectionId,
	}).FirstOrCreate(assignment)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) GetBenchmarkAssignmentsByConnectionId(ctx context.Context, connectionId string) ([]BenchmarkAssignment, error) {
	var s []BenchmarkAssignment
	tx := db.Orm.WithContext(ctx).Model(&BenchmarkAssignment{}).
		Where(BenchmarkAssignment{ConnectionId: &connectionId}).
		Where("resource_collection IS NULL").Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) GetBenchmarkAssignmentsByResourceCollectionId(ctx context.Context, resourceCollectionId string) ([]BenchmarkAssignment, error) {
	var s []BenchmarkAssignment
	tx := db.Orm.WithContext(ctx).Model(&BenchmarkAssignment{}).
		Where(BenchmarkAssignment{ResourceCollection: &resourceCollectionId}).
		Where("connection_id IS NULL").Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) GetBenchmarkAssignmentsByBenchmarkId(ctx context.Context, benchmarkId string) ([]BenchmarkAssignment, error) {
	var s []BenchmarkAssignment
	tx := db.Orm.WithContext(ctx).Model(&BenchmarkAssignment{}).Where(BenchmarkAssignment{BenchmarkId: benchmarkId}).Scan(&s)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) ListBenchmarkAssignments(ctx context.Context) ([]BenchmarkAssignment, error) {
	var s []BenchmarkAssignment
	tx := db.Orm.WithContext(ctx).Model(&BenchmarkAssignment{}).Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) GetBenchmarkAssignmentByIds(ctx context.Context, benchmarkId string, connectionId, resourceCollectionId *string) (*BenchmarkAssignment, error) {
	var s BenchmarkAssignment
	tx := db.Orm.WithContext(ctx).Model(&BenchmarkAssignment{}).Where(BenchmarkAssignment{
		BenchmarkId:        benchmarkId,
		ConnectionId:       connectionId,
		ResourceCollection: resourceCollectionId,
	}).Scan(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &s, nil
}

func (db Database) DeleteBenchmarkAssignmentByIds(ctx context.Context, benchmarkId string, connectionId, resourceCollectionId *string) error {
	tx := db.Orm.WithContext(ctx).Unscoped().Where(BenchmarkAssignment{
		BenchmarkId:        benchmarkId,
		ConnectionId:       connectionId,
		ResourceCollection: resourceCollectionId,
	}).Delete(&BenchmarkAssignment{})

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) DeleteBenchmarkAssignmentByBenchmarkId(ctx context.Context, benchmarkId string) error {
	tx := db.Orm.WithContext(ctx).Unscoped().Where(BenchmarkAssignment{BenchmarkId: benchmarkId}).Delete(&BenchmarkAssignment{})

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) ListComplianceTagKeysWithPossibleValues(ctx context.Context) (map[string][]string, error) {
	var tags []BenchmarkTag
	tx := db.Orm.WithContext(ctx).Model(BenchmarkTag{}).Find(&tags)
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

func (db Database) ListInsightTagKeysWithPossibleValues(ctx context.Context) (map[string][]string, error) {
	var tags []InsightTag
	tx := db.Orm.WithContext(ctx).Model(InsightTag{}).Find(&tags)
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

func (db Database) GetInsightTagTagPossibleValues(ctx context.Context, key string) ([]string, error) {
	var tags []InsightTag
	tx := db.Orm.WithContext(ctx).Model(InsightTag{}).Where("key = ?", key).Find(&tags)
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

func (db Database) GetInsight(ctx context.Context, id uint) (*Insight, error) {
	var res Insight
	tx := db.Orm.WithContext(ctx).Model(&Insight{}).Preload(clause.Associations).
		Where("id = ?", id).
		First(&res)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &res, nil
}

func (db Database) ListInsightsWithFilters(ctx context.Context, insightIDs []uint, connectors []source.Type, enabled *bool, tags map[string][]string) ([]Insight, error) {
	var s []Insight
	m := db.Orm.WithContext(ctx).Model(&Insight{}).Preload(clause.Associations)
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

func (db Database) ListInsightGroups(ctx context.Context, connectors []source.Type, tags map[string][]string) ([]InsightGroup, error) {
	var insightGroups []InsightGroup
	m := db.Orm.WithContext(ctx).Model(&InsightGroup{}).Preload(clause.Associations).Find(&insightGroups)
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
	insights, err := db.ListInsightsWithFilters(ctx, insightIDs, nil, nil, nil)
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

func (db Database) GetInsightGroup(ctx context.Context, id uint) (*InsightGroup, error) {
	var res InsightGroup
	tx := db.Orm.WithContext(ctx).Model(&InsightGroup{}).Preload(clause.Associations).
		Where("id = ?", id).
		First(&res)
	if tx.Error != nil {
		return nil, tx.Error
	}
	insightIDs := make([]uint, 0, len(res.Insights))
	for _, insight := range res.Insights {
		insightIDs = append(insightIDs, insight.ID)
	}
	insights, err := db.ListInsightsWithFilters(ctx, insightIDs, nil, nil, nil)
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

func (db Database) ListControls(ctx context.Context, controlIDs []string, tags map[string][]string) ([]Control, error) {
	var s []Control
	tx := db.Orm.WithContext(ctx).Model(&Control{}).Preload(clause.Associations)
	if len(controlIDs) > 0 {
		tx = tx.Where("id IN ?", controlIDs)
	}
	if len(tags) > 0 {
		tx = tx.Joins("JOIN control_tags AS tags ON tags.control_id = controls.id")
		for key, values := range tags {
			if len(values) > 0 {
				tx = tx.Where("tags.key = ? AND (tags.value && ?)", key, pq.StringArray(values))
			} else {
				tx = tx.Where("tags.key = ?", key)
			}
		}
	}
	tx = tx.Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) ListQueries(ctx context.Context) ([]Query, error) {
	var s []Query
	tx := db.Orm.WithContext(ctx).Model(&Query{}).Preload(clause.Associations).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) ListControlsBare(ctx context.Context) ([]Control, error) {
	var s []Control
	tx := db.Orm.WithContext(ctx).Model(&Control{}).Preload("Tags").
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}
