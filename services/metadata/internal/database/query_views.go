package database

import "github.com/opengovern/opencomply/services/metadata/models"

func (db Database) ListQueryViews() ([]models.QueryView, error) {
	var queryViews []models.QueryView
	err := db.orm.Model(&models.QueryView{}).Find(&queryViews).Error
	if err != nil {
		return nil, err
	}
	return queryViews, nil
}
