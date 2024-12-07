package database

import (
	"github.com/opengovern/opencomply/services/metadata/models"
	"gorm.io/gorm/clause"
)

func (db Database) ListQueryViews() ([]models.QueryView, error) {
	var queryViews []models.QueryView
	err := db.orm.
		Model(&models.QueryView{}).
		Preload(clause.Associations).
		Find(&queryViews).Error
	if err != nil {
		return nil, err
	}
	return queryViews, nil
}
