package db

import (
	"github.com/lib/pq"
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/og-util/pkg/model"
	"gorm.io/gorm/clause"
)

type MetricType string

const (
	MetricTypeAssets MetricType = "assets"
	MetricTypeSpend  MetricType = "spend"
)

type MetricTag struct {
	model.Tag
	ID string `gorm:"primaryKey; type:citext"`
}

type QueryEngine string

const (
	QueryEngine_cloudql     QueryEngine = "cloudql"
	QueryEngine_cloudqlRego QueryEngine = "cloudql-rego"
	QueryEngine_NotDefined  QueryEngine = ""
)

type AnalyticMetricStatus string

const (
	AnalyticMetricStatusActive    AnalyticMetricStatus = "active"
	AnalyticMetricStatusInvisible AnalyticMetricStatus = "invisible"
	AnalyticMetricStatusInactive  AnalyticMetricStatus = "inactive"
)

type AnalyticMetric struct {
	ID                       string `gorm:"primaryKey"`
	Engine                   QueryEngine
	IntegrationTypes         pq.StringArray `gorm:"type:text[]"`
	Type                     MetricType
	Name                     string
	Query                    string
	Tables                   pq.StringArray `gorm:"type:text[]"`
	FinderQuery              string
	FinderPerConnectionQuery string
	Visible                  bool
	Status                   AnalyticMetricStatus
	Tags                     []MetricTag         `gorm:"foreignKey:ID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap                  map[string][]string `gorm:"-:all"`
}

func (m *AnalyticMetric) GetTagsMap() map[string][]string {
	if m.tagsMap != nil {
		return m.tagsMap
	}

	m.tagsMap = map[string][]string{}
	for _, tag := range m.Tags {
		m.tagsMap[tag.GetKey()] = tag.GetValue()
	}
	return m.tagsMap
}

func (db Database) ListMetrics(statuses []AnalyticMetricStatus) ([]AnalyticMetric, error) {
	var s []AnalyticMetric
	tx := db.orm.Model(AnalyticMetric{}).Preload(clause.Associations)
	if len(statuses) > 0 {
		tx = tx.Where("analytic_metrics.status IN ?", statuses)
	}
	tx = tx.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) GetMetricByID(metricID string) (*AnalyticMetric, error) {
	var s *AnalyticMetric

	tx := db.orm.Model(AnalyticMetric{}).Preload(clause.Associations).Where("id = ?", metricID).Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) GetMetric(metricType MetricType, table string) (*AnalyticMetric, error) {
	var s *AnalyticMetric
	tx := db.orm.Model(AnalyticMetric{}).Preload(clause.Associations).Where("type = ?", metricType).Where("? = ANY (tables)", table).Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) ListFilteredMetrics(tags map[string][]string, metricType MetricType,
	metricIDs []string, integrationTypes []integration.Type, statuses []AnalyticMetricStatus) ([]AnalyticMetric, error) {
	var metrics []AnalyticMetric
	query := db.orm.Model(AnalyticMetric{}).Preload(clause.Associations)
	if len(tags) > 0 {
		query = query.Joins("JOIN metric_tags AS tags ON tags.id = analytic_metrics.id")
		for key, values := range tags {
			if len(values) > 0 {
				query = query.Where("tags.key = ? AND (tags.value && ?)", key, pq.StringArray(values))
			} else {
				query = query.Where("tags.key = ?", key)
			}
		}
	}
	if len(integrationTypes) > 0 {
		for _, ct := range integrationTypes {
			query = query.Where("? = ANY (analytic_metrics.connectors)", ct)
		}
	}
	if len(statuses) > 0 {
		query = query.Where("analytic_metrics.status IN ?", statuses)
	}
	if len(metricIDs) > 0 {
		query = query.Where("analytic_metrics.id IN ?", metricIDs)
	}
	if len(metricType) > 0 {
		query = query.Where("type = ?", metricType)
	}
	tx := query.Find(&metrics)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return metrics, nil
}

func (db Database) ListMetricTagsKeysWithPossibleValues(integrationTypes []integration.Type) (map[string][]string, error) {
	var tags []MetricTag
	tx := db.orm.Model(MetricTag{}).Joins("JOIN analytic_metrics ON metric_tags.id = analytic_metrics.id")
	if len(integrationTypes) > 0 {
		for _, ct := range integrationTypes {
			tx = tx.Where("? = ANY (analytic_metrics.integration_types)", ct)
		}
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
