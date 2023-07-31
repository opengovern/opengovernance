package db

import (
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/lib/pq"
	"gorm.io/gorm/clause"
)

type MetricTag struct {
	model.Tag
	ID string `gorm:"primaryKey; type:citext"`
}

type AnalyticMetric struct {
	ID          string         `gorm:"primaryKey"`
	Connectors  pq.StringArray `gorm:"type:text[]"`
	Name        string
	Query       string
	Tables      []string
	FinderQuery string
	Tags        []MetricTag         `gorm:"foreignKey:ID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap     map[string][]string `gorm:"-:all"`
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

func (db Database) ListMetrics() ([]AnalyticMetric, error) {
	var s []AnalyticMetric
	tx := db.orm.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) ListFilteredMetrics(tags map[string][]string, metricIDs []string, connectorTypes []source.Type) ([]AnalyticMetric, error) {
	var metrics []AnalyticMetric
	query := db.orm.Model(AnalyticMetric{}).Preload(clause.Associations)
	if len(tags) != 0 {
		query = query.Joins("JOIN metric_tags AS tags ON tags.id = analytic_metrics.id")
		for key, values := range tags {
			if len(values) != 0 {
				query = query.Where("tags.key = ? AND (tags.value && ?)", key, pq.StringArray(values))
			} else {
				query = query.Where("tags.key = ?", key)
			}
		}
	}
	if len(connectorTypes) > 0 {
		for _, ct := range connectorTypes {
			query = query.Where("? = ANY (analytic_metrics.connectors)", ct)
		}
	}
	if len(metricIDs) != 0 {
		query = query.Where("analytic_metrics.id IN ?", metricIDs)
	}
	tx := query.Find(&metrics)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return metrics, nil
}

func (db Database) ListMetricTagsKeysWithPossibleValues(connectorTypes []source.Type) (map[string][]string, error) {
	var tags []MetricTag
	tx := db.orm.Model(MetricTag{}).Joins("JOIN analytic_metrics ON metric_tags.id = analytic_metrics.id")
	if len(connectorTypes) > 0 {
		for _, ct := range connectorTypes {
			tx = tx.Where("? = ANY (analytic_metrics.connectors)", ct)
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
