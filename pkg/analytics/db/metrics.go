package db

import "github.com/lib/pq"

type Metric struct {
	ID         string         `gorm:"primaryKey"`
	Connectors pq.StringArray `gorm:"type:text[]"`
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
