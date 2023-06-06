package workspace

import (
	"encoding/json"
	"gitlab.com/keibiengine/keibi-engine/pkg/metadata/models"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/db"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
	"os"
)

func Run(db db.Database, logger *zap.Logger, wsFolder string) error {
	if err := OnboardMigration(db, logger, wsFolder+"/onboard.json"); err != nil {
		return err
	}
	if err := MetadataMigration(db, logger, wsFolder+"/metadata.json"); err != nil {
		return err
	}
	if err := InventoryMigration(db, logger, wsFolder+"/inventory.json"); err != nil {
		return err
	}
	return nil
}

func OnboardMigration(db db.Database, logger *zap.Logger, onboardFilePath string) error {
	content, err := os.ReadFile(onboardFilePath)
	if err != nil {
		return err
	}

	var connectors []onboard.Connector
	err = json.Unmarshal(content, &connectors)
	if err != nil {
		return err
	}

	for _, obj := range connectors {
		err := db.ORM.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "name"}}, // key colume
			DoUpdates: clause.AssignmentColumns([]string{"label", "short_description", "description", "direction",
				"status", "logo", "auto_onboard_support", "allow_new_connections", "max_connection_limit", "tags"}),
		}).Create(&obj).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func MetadataMigration(db db.Database, logger *zap.Logger, metadataFilePath string) error {
	content, err := os.ReadFile(metadataFilePath)
	if err != nil {
		return err
	}

	var metadata []models.ConfigMetadata
	err = json.Unmarshal(content, &metadata)
	if err != nil {
		return err
	}

	for _, obj := range metadata {
		err := db.ORM.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}}, // key colume
			DoUpdates: clause.AssignmentColumns([]string{"type", "value"}),
		}).Create(&obj).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func InventoryMigration(db db.Database, logger *zap.Logger, onboardFilePath string) error {
	return nil
}
