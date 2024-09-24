package inventory

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	analyticsDb "github.com/kaytu-io/open-governance/pkg/analytics/db"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Database struct {
	orm *gorm.DB
}

func NewDatabase(orm *gorm.DB) Database {
	return Database{orm: orm}
}

func (db Database) Initialize() error {
	err := db.orm.AutoMigrate(
		&Query{},
		&QueryParameter{},
		&ResourceType{},
		&NamedQuery{},
		&NamedQueryTag{},
		&NamedQueryHistory{},
		&ResourceTypeTag{},
		&analyticsDb.AnalyticMetric{},
		&analyticsDb.MetricTag{},
		&ResourceCollection{},
		&ResourceCollectionTag{},
		&ResourceTypeV2{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (db Database) GetQueriesWithFilters(search *string) ([]NamedQuery, error) {
	var s []NamedQuery

	m := db.orm.Model(&NamedQuery{})

	if search != nil {
		m = m.Where("title like ?", "%"+*search+"%")
	}
	tx := m.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	v := map[string]NamedQuery{}
	for _, item := range s {
		if _, ok := v[item.ID]; !ok {
			v[item.ID] = item
		}
	}
	var res []NamedQuery
	for _, val := range v {
		res = append(res, val)
	}

	for i, sq := range res {
		if sq.QueryID != nil {
			var query Query
			tx := db.orm.Model(&Query{}).Preload(clause.Associations).Where("id = ?", *sq.QueryID).First(&query)
			if tx.Error != nil {
				return nil, tx.Error
			}
			res[i].Query = &query
		}
	}

	return res, nil
}

func (db Database) ListQueries(queryIds []string, tables map[string]bool) ([]NamedQuery, error) {
	var s []NamedQuery

	m := db.orm.Model(&NamedQuery{})

	if len(queryIds) > 0 {
		m = m.Where("id in ?", queryIds)
	}

	tx := m.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	v := map[string]NamedQuery{}
	for _, item := range s {
		if _, ok := v[item.ID]; !ok {
			v[item.ID] = item
		}
	}
	var res []NamedQuery
	for _, val := range v {
		res = append(res, val)
	}

	for i, sq := range res {
		if sq.QueryID != nil {
			var query Query
			tx := db.orm.Model(&Query{}).Preload(clause.Associations).Where("id = ?", *sq.QueryID).First(&query)
			if tx.Error != nil {
				return nil, tx.Error
			}
			exists := false
			for _, t := range query.ListOfTables {
				if _, ok := tables[t]; ok && len(tables) > 0 {
					exists = true
				}
			}
			if exists {
				res[i].Query = &query

			}
		}
	}

	return res, nil
}

func (db Database) GetQuery(id string) (*NamedQuery, error) {
	var s NamedQuery
	tx := db.orm.Model(NamedQuery{}).Preload(clause.Associations).Preload("Tags").Where("id = ?", id).First(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}

	if s.QueryID != nil {
		var query Query
		tx := db.orm.Model(&Query{}).Preload(clause.Associations).Where("id = ?", *s.QueryID).First(&query)
		if tx.Error != nil {
			return nil, tx.Error
		}
		s.Query = &query
	}

	return &s, nil
}

func (db Database) GetQueriesWithTagsFilters(search *string, tagFilters map[string][]string, connectors []string) ([]NamedQuery, error) {
	var s []NamedQuery

	m := db.orm.Model(&NamedQuery{}).Preload(clause.Associations).Preload("Tags")

	if search != nil {
		m = m.Where("title LIKE ?", "%"+*search+"%")
	}

	for i, c := range connectors {
		connectors[i] = strings.ToLower(c)
	}

	if len(connectors) > 0 {
		m = m.Where("named_queries.connector::text[] @> ?", pq.Array(connectors))
	}

	if len(tagFilters) > 0 {
		i := 0
		for key, values := range tagFilters {
			alias := fmt.Sprintf("t%d", i)
			joinCondition := fmt.Sprintf("JOIN named_query_tags %s ON %s.named_query_id = named_queries.id", alias, alias)

			m = m.Joins(joinCondition).Where(fmt.Sprintf("%s.key = ? AND %s.value::text[] @> ?", alias, alias), key, pq.Array(values))

			i++
		}
	}

	tx := m.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	v := map[string]NamedQuery{}
	for _, item := range s {
		if _, ok := v[item.ID]; !ok {
			v[item.ID] = item
		}
	}

	var res []NamedQuery
	for _, val := range v {
		res = append(res, val)
	}

	for i, sq := range res {
		if sq.QueryID != nil {
			var query Query
			tx := db.orm.Model(&Query{}).Preload(clause.Associations).Where("id = ?", *sq.QueryID).First(&query)
			if tx.Error != nil {
				return nil, tx.Error
			}
			res[i].Query = &query
		}
	}

	return res, nil
}

func (db Database) GetQueriesTags() ([]NamedQueryTagsResult, error) {
	var results []NamedQueryTagsResult

	// Execute the raw SQL query
	query := `SELECT 
    key, 
    ARRAY_AGG(DISTINCT value) AS unique_values
FROM (
    SELECT 
        key, 
        UNNEST(value) AS value
    FROM named_query_tags
) AS expanded_values
GROUP BY key;
`
	err := db.orm.Raw(query).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (db Database) GetQueryHistory() ([]NamedQueryHistory, error) {
	var history []NamedQueryHistory
	tx := db.orm.Order("executed_at desc").Limit(3).Find(&history)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return history, nil
}

func (db Database) UpdateQueryHistory(query string) error {
	history := NamedQueryHistory{
		Query:      query,
		ExecutedAt: time.Now(),
	}
	// Upsert query history
	err := db.orm.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "query"}},
		DoUpdates: clause.AssignmentColumns([]string{"executed_at"}),
	}).Create(&history).Error
	if err != nil {
		return err
	}

	// Only keep latest 100 queries in history
	const keepNumber = 100
	var count int64
	err = db.orm.Model(&NamedQueryHistory{}).Count(&count).Error
	if err != nil {
		return err
	}
	if count > keepNumber {
		var oldest NamedQueryHistory
		err = db.orm.Model(&NamedQueryHistory{}).Order("executed_at desc").Offset(keepNumber - 1).Limit(1).Find(&oldest).Error
		if err != nil {
			return err
		}

		err = db.orm.Model(&NamedQueryHistory{}).Where("executed_at < ?", oldest.ExecutedAt).Delete(&NamedQueryHistory{}).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (db Database) ListResourceTypeTagsKeysWithPossibleValues(connectorTypes []source.Type, doSummarize *bool) (map[string][]string, error) {
	var tags []ResourceTypeTag
	tx := db.orm.Model(ResourceTypeTag{}).Joins("JOIN resource_types ON resource_type_tags.resource_type = resource_types.resource_type")
	if doSummarize != nil {
		tx = tx.Where("resource_types.do_summarize = ?", true)
	}
	if len(connectorTypes) > 0 {
		tx = tx.Where("resource_types.connector in ?", connectorTypes)
	}
	tx.Find(&tags)
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

func (db Database) ListFilteredResourceTypes(tags map[string][]string, resourceTypeNames []string, serviceNames []string, connectorTypes []source.Type, doSummarize bool) ([]ResourceType, error) {
	var resourceTypes []ResourceType
	query := db.orm.Model(ResourceType{}).Preload(clause.Associations)
	if doSummarize {
		query = query.Where("resource_types.do_summarize = ?", doSummarize)
	}
	if len(tags) != 0 {
		query = query.Joins("JOIN resource_type_tags AS tags ON tags.resource_type = resource_types.resource_type")
		for key, values := range tags {
			if len(values) != 0 {
				query = query.Where("tags.key = ? AND (tags.value && ?)", key, pq.StringArray(values))
			} else {
				query = query.Where("tags.key = ?", key)
			}
		}
	}
	if len(serviceNames) != 0 {
		query = query.Where("service_name IN ?", serviceNames)
	}
	if len(connectorTypes) != 0 {
		query = query.Where("connector IN ?", connectorTypes)
	}
	if len(resourceTypeNames) != 0 {
		query = query.Where("resource_types.resource_type IN ?", resourceTypeNames)
	}
	tx := query.Find(&resourceTypes)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return resourceTypes, nil
}

func (db Database) GetResourceType(resourceType string) (*ResourceType, error) {
	var rtObj ResourceType
	tx := db.orm.Model(ResourceType{}).Preload(clause.Associations).Where("resource_type = ?", resourceType).First(&rtObj)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &rtObj, nil
}

func (db Database) ListResourceCollections(ids []string, statuses []ResourceCollectionStatus) ([]ResourceCollection, error) {
	var resourceCollections []ResourceCollection
	tx := db.orm.Model(ResourceCollection{}).Preload(clause.Associations)
	if len(ids) > 0 {
		tx = tx.Where("id IN ?", ids)
	}
	if len(statuses) > 0 {
		tx = tx.Where("status IN ?", statuses)
	}
	tx.Find(&resourceCollections)
	if tx.Error != nil {
		return nil, tx.Error
	}
	for i := range resourceCollections {
		if len(resourceCollections[i].FiltersJson.Bytes) > 0 {
			err := json.Unmarshal(resourceCollections[i].FiltersJson.Bytes, &resourceCollections[i].Filters)
			if err != nil {
				return nil, err
			}
		}
	}

	return resourceCollections, nil
}

func (db Database) GetResourceCollection(collectionID string) (*ResourceCollection, error) {
	var collection ResourceCollection
	tx := db.orm.Model(ResourceCollection{}).Preload(clause.Associations).Where("id = ?", collectionID).First(&collection)
	if tx.Error != nil {
		return nil, tx.Error
	}

	if len(collection.FiltersJson.Bytes) > 0 {
		err := json.Unmarshal(collection.FiltersJson.Bytes, &collection.Filters)
		if err != nil {
			return nil, err
		}
	}

	return &collection, nil
}

func (db Database) ListNamedQueriesUniqueProviders() ([]string, error) {
	var connectors []string

	tx := db.orm.
		Model(&NamedQuery{}).
		Select("DISTINCT UNNEST(connectors)").
		Scan(&connectors)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return connectors, nil
}

func (db Database) ListResourceTypesUniqueCategories() ([]string, error) {
	var connectors []string

	tx := db.orm.
		Model(&ResourceTypeV2{}).
		Select("DISTINCT category").
		Scan(&connectors)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return connectors, nil
}

func (db Database) ListCategoryResourceTypes(category string) ([]ResourceTypeV2, error) {
	var resourceTypes []ResourceTypeV2

	tx := db.orm.
		Model(&ResourceTypeV2{}).
		Where("category = ?", category)

	tx = tx.Find(&resourceTypes)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return resourceTypes, nil
}

func (db Database) ListUniqueCategoriesAndTablesForTables(tables []string) ([]CategoriesTables, error) {
	var results []CategoriesTables

	query := `
        SELECT 
            category, 
            ARRAY_AGG(steampipe_table ORDER BY steampipe_table) AS tables
        FROM 
            resource_type_v2`

	if len(tables) > 0 {
		query = query + ` WHERE steampipe_table in ?`
	}
	query = query + ` GROUP BY category`

	err := db.orm.Raw(query, tables).Scan(&results).Error

	if err != nil {
		return nil, err
	}
	return results, nil
}
