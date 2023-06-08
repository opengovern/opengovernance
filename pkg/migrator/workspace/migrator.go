package workspace

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"gitlab.com/keibiengine/keibi-engine/pkg/metadata/models"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/db"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
	"os"
)

func Run(conf migrator.JobConfig, logger *zap.Logger, wsFolder string) error {
	if err := OnboardMigration(conf, logger, wsFolder+"/onboard.json"); err != nil {
		return err
	}
	if err := MetadataMigration(conf, logger, wsFolder+"/metadata.json"); err != nil {
		return err
	}
	if err := InventoryMigration(conf, logger, wsFolder+"/inventory.json"); err != nil {
		return err
	}
	return nil
}

func OnboardMigration(conf migrator.JobConfig, logger *zap.Logger, onboardFilePath string) error {
	cfg := postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      "onboard",
		SSLMode: conf.PostgreSQL.SSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return fmt.Errorf("new postgres client: %w", err)
	}
	dbm := db.Database{ORM: orm}

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
		err := dbm.ORM.Clauses(clause.OnConflict{
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

func MetadataMigration(conf migrator.JobConfig, logger *zap.Logger, metadataFilePath string) error {
	cfg := postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      "metadata",
		SSLMode: conf.PostgreSQL.SSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return fmt.Errorf("new postgres client: %w", err)
	}
	dbm := db.Database{ORM: orm}

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
		err := dbm.ORM.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}}, // key colume
			DoUpdates: clause.AssignmentColumns([]string{"type", "value"}),
		}).Create(&obj).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func InventoryMigration(db migrator.JobConfig, logger *zap.Logger, onboardFilePath string) error {
	return nil
}
