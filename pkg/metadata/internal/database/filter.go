package database

import (
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
)

func (db Database) AddFilter(filter models.Filter) error {
	return db.orm.Model(&models.Filter{}).Create(filter).Error
}

func (db Database) ListFilters() ([]models.Filter, error) {
	var filters []models.Filter
	err := db.orm.Model(&models.Filter{}).First(&filters).Error
	if err != nil {
		return nil, err
	}
	return filters, nil
}
