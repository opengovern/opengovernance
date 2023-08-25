package database

import (
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"

	"gorm.io/gorm/clause"
)

func (db DatabaseFilter) upsetConfigFilter(configFilter models.Filters) error {
	return db.orm.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}, {Name: "keyValue"}},
	}).Create(&configFilter).Error
}

func (db DatabaseFilter) SetListFilters(filters models.Filters) error {
	return db.upsetConfigFilter(models.Filters{
		Name:     filters.Name,
		KeyValue: filters.KeyValue,
	})
}

func (db DatabaseFilter) GetListFilters(name string) (models.Filters, error) {
	var configFilter models.Filters
	err := db.orm.First(&configFilter, "name = ?", name).Error
	if err != nil {
		return models.Filters{}, err
	}
	return configFilter, nil
}
