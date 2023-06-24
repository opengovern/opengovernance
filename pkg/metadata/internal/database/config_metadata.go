package database

import (
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	"gorm.io/gorm/clause"
)

func (db Database) upsertConfigMetadata(configMetadata models.ConfigMetadata) error {
	return db.orm.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "type"}),
	}).Create(&configMetadata).Error
}

func (db Database) SetConfigMetadata(cm models.ConfigMetadata) error {
	return db.upsertConfigMetadata(models.ConfigMetadata{
		Key:   cm.Key,
		Type:  cm.Type,
		Value: cm.Value,
	})
}

func (db Database) GetConfigMetadata(key string) (models.IConfigMetadata, error) {
	var configMetadata models.ConfigMetadata
	err := db.orm.First(&configMetadata, "key = ?", key).Error
	if err != nil {
		return nil, err
	}
	return configMetadata.ParseToType()
}
