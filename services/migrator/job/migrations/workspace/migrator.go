package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/open-governance/pkg/metadata/models"
	"github.com/kaytu-io/open-governance/services/integration/model"
	"github.com/kaytu-io/open-governance/services/migrator/config"
	"github.com/kaytu-io/open-governance/services/migrator/db"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
	"os"
)

type Migration struct {
}

func (m Migration) AttachmentFolderPath() string {
	return "/workspace-migration"
}

func (m Migration) IsGitBased() bool {
	return false
}

func (m Migration) Run(ctx context.Context, conf config.MigratorConfig, logger *zap.Logger) error {
	if err := OnboardMigration(conf, logger, m.AttachmentFolderPath()+"/onboard.json"); err != nil {
		logger.Fatal("onboard migration failed", zap.Error(err))
		return err
	}
	if err := MetadataMigration(conf, logger, m.AttachmentFolderPath()+"/metadata.json"); err != nil {
		return err
	}
	if err := MetadataQueryParamMigration(conf, logger, m.AttachmentFolderPath()+"/query_parameters.json"); err != nil {
		return err
	}
	return nil
}

func OnboardMigration(conf config.MigratorConfig, logger *zap.Logger, onboardFilePath string) error {
	orm, err := postgres.NewClient(&postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      "onboard",
		SSLMode: conf.PostgreSQL.SSLMode,
	}, logger)
	if err != nil {
		return fmt.Errorf("new postgres client: %w", err)
	}
	dbm := db.Database{ORM: orm}

	content, err := os.ReadFile(onboardFilePath)
	if err != nil {
		return err
	}

	logger.Info("connectors json:", zap.String("json", string(content)))

	var connectors []model.Connector
	err = json.Unmarshal(content, &connectors)
	if err != nil {
		return err
	}

	for _, obj := range connectors {
		logger.Info("connector", zap.Any("obj", obj))
		err := dbm.ORM.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "name"}}, // key colume
			DoUpdates: clause.AssignmentColumns([]string{"id", "label", "short_description", "description", "direction",
				"status", "logo", "auto_onboard_support", "allow_new_connections", "max_connection_limit", "tags", "tier"}),
		}).Create(&obj).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func MetadataMigration(conf config.MigratorConfig, logger *zap.Logger, metadataFilePath string) error {
	orm, err := postgres.NewClient(&postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      "metadata",
		SSLMode: conf.PostgreSQL.SSLMode,
	}, logger)
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
			DoNothing: true,
		}).Create(&obj).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func MetadataQueryParamMigration(conf config.MigratorConfig, logger *zap.Logger, metadataQueryParamFilePath string) error {
	orm, err := postgres.NewClient(&postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      "metadata",
		SSLMode: conf.PostgreSQL.SSLMode,
	}, logger)
	if err != nil {
		return fmt.Errorf("new postgres client: %w", err)
	}
	dbm := db.Database{ORM: orm}

	content, err := os.ReadFile(metadataQueryParamFilePath)
	if err != nil {
		return err
	}

	var queryParameters []models.QueryParameter
	err = json.Unmarshal(content, &queryParameters)
	if err != nil {
		return err
	}

	for _, obj := range queryParameters {
		err := dbm.ORM.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&obj).Error
		if err != nil {
			return err
		}
	}
	return nil
}
