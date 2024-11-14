package database

import (
	"errors"

	"github.com/opengovern/opengovernance/services/metadata/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (db Database) upsertQueryParameter(queryParam models.QueryParameter) error {
	return db.orm.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&queryParam).Error
}

func (db Database) upsertQueryParameters(queryParam []*models.QueryParameter) error {
	return db.orm.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(queryParam).Error
}

func (db Database) SetQueryParameter(key string, value string) error {
	return db.upsertQueryParameter(models.QueryParameter{
		Key:   key,
		Value: value,
	})
}

func (db Database) SetQueryParameters(queryParams []*models.QueryParameter) error {
	return db.upsertQueryParameters(queryParams)
}

func (db Database) GetQueryParameter(key string) (*models.QueryParameter, error) {
	var queryParam models.QueryParameter
	err := db.orm.First(&queryParam, "key = ?", key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &queryParam, nil
}

func (db Database) GetQueryParameters() ([]models.QueryParameter, error) {
	var queryParams []models.QueryParameter
	err := db.orm.Find(&queryParams).Error
	if err != nil {
		return nil, err
	}
	return queryParams, nil
}

func (db Database) DeleteQueryParameter(key string) error {
	return db.orm.Unscoped().Delete(&models.QueryParameter{}, "key = ?", key).Error
}
