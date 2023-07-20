package db

import (
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/lib/pq"
	"gorm.io/gorm/clause"
)

type MetricTag struct {
	model.Tag
	Name string `gorm:"primaryKey; type:citext"`
}

type AnalyticMetric struct {
	ID         string         `gorm:"primaryKey"`
	Connectors pq.StringArray `gorm:"type:text[]"`
	Name       string
	Query      string
	Tags       []MetricTag         `gorm:"foreignKey:Name;references:Name;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	tagsMap    map[string][]string `gorm:"-:all"`
}

func (db Database) ListMetrics() ([]AnalyticMetric, error) {
	var s []AnalyticMetric
	tx := db.orm.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) ListFilteredMetrics(tags map[string][]string, metricNames []string, connectorTypes []source.Type) ([]AnalyticMetric, error) {
	var metrics []AnalyticMetric
	query := db.orm.Model(AnalyticMetric{}).Preload(clause.Associations)
	if len(tags) != 0 {
		query = query.Joins("JOIN metric_tags AS tags ON tags.name = analytic_metrics.name")
		for key, values := range tags {
			if len(values) != 0 {
				query = query.Where("tags.key = ? AND (tags.value && ?)", key, pq.StringArray(values))
			} else {
				query = query.Where("tags.key = ?", key)
			}
		}
	}
	if len(connectorTypes) != 0 {
		query = query.Where("connector IN ?", connectorTypes)
	}
	if len(metricNames) != 0 {
		query = query.Where("analytic_metrics.name IN ?", metricNames)
	}
	tx := query.Find(&metrics)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return metrics, nil
}
