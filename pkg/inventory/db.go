package inventory

import (
	"github.com/jackc/pgx/v4"
	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
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
		&Service{},
		&ResourceType{},
		&SmartQuery{},
		&Metric{},
		&ResourceTypeTag{},
		&ServiceTag{},
	)
	if err != nil {
		return err
	}

	awsResourceTypes := aws.GetResourceTypesMap()
	for _, resourceType := range awsResourceTypes {
		err = db.orm.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&Service{
			ServiceName: resourceType.ServiceName,
			Connector:   source.CloudAWS,
		}).Error
		if err != nil {
			return err
		}
		err = db.orm.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&ResourceType{
			Connector:     source.CloudAWS,
			ResourceType:  resourceType.ResourceName,
			ResourceLabel: resourceType.ResourceLabel,
			ServiceName:   resourceType.ServiceName,
		}).Error
		if err != nil {
			return err
		}
	}
	azureResourceTypes := azure.GetResourceTypesMap()
	for _, resourceType := range azureResourceTypes {
		err = db.orm.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&Service{
			ServiceName: resourceType.ServiceName,
			Connector:   source.CloudAzure,
		}).Error
		if err != nil {
			return err
		}
		err = db.orm.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&ResourceType{
			Connector:     source.CloudAzure,
			ResourceType:  resourceType.ResourceName,
			ResourceLabel: resourceType.ResourceLabel,
			ServiceName:   resourceType.ServiceName,
		}).Error
		if err != nil {
			return err
		}
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
		if _, ok := v[item.ID]; !ok {
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

func (db Database) CreateOrUpdateMetric(metric Metric) error {
	return db.orm.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source_id"}, {Name: "resource_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"schedule_job_id", "count", "last_day_count", "last_week_count", "last_quarter_count", "last_year_count"}),
	}).Create(metric).Error
}

func (db Database) FetchConnectionMetrics(sourceID string, resourceTypes []string) ([]Metric, error) {
	var metrics []Metric
	tx := db.orm.Model(Metric{}).
		Where("source_id = ?", sourceID).
		Where("resource_type in ?", resourceTypes).
		Find(&metrics)
	return metrics, tx.Error
}

func (db Database) FetchConnectionAllMetrics(sourceID string) ([]Metric, error) {
	var metrics []Metric
	tx := db.orm.Model(Metric{}).
		Where("source_id = ?", sourceID).
		Find(&metrics)
	return metrics, tx.Error
}

func (db Database) FetchProviderAllMetrics(provider source.Type) ([]Metric, error) {
	var metrics []Metric
	tx := db.orm.Model(Metric{}).
		Where("provider = ?", string(provider)).
		Find(&metrics)
	return metrics, tx.Error
}

func (db Database) FetchProviderMetrics(provider source.Type, resourceTypes []string) ([]Metric, error) {
	var metrics []Metric
	tx := db.orm.Model(Metric{}).
		Where("provider = ?", string(provider)).
		Where("resource_type in ?", resourceTypes).
		Find(&metrics)
	return metrics, tx.Error
}

func (db Database) FetchMetrics(resourceTypes []string) ([]Metric, error) {
	var metrics []Metric
	tx := db.orm.Model(Metric{}).
		Where("resource_type in ?", resourceTypes).
		Find(&metrics)
	return metrics, tx.Error
}

func (db Database) ListMetrics() ([]Metric, error) {
	var metrics []Metric
	tx := db.orm.Model(Metric{}).Find(&metrics)
	return metrics, tx.Error
}
