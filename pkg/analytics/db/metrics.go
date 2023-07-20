package db

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Metric struct {
	ID         string `gorm:"primaryKey"`
	Connectors []source.Type
	Name       string
	Query      string
}

func (db Database) ListMetrics() ([]Metric, error) {
	var s []Metric
	tx := db.orm.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}
