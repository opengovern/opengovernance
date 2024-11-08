package integration

import (
	"context"
	"encoding/json"
	"fmt"
	integrationModels "github.com/opengovern/opengovernance/services/integration/models"
	"github.com/opengovern/opengovernance/services/migrator/config"
	"github.com/opengovern/opengovernance/services/migrator/db"
	"gorm.io/gorm/clause"
	"os"

	"github.com/opengovern/og-util/pkg/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m Migration) IsGitBased() bool {
	return true
}
func (m Migration) AttachmentFolderPath() string {
	return config.IntegrationGroupsGitPath
}

func (m Migration) Run(ctx context.Context, conf config.MigratorConfig, logger *zap.Logger) error {
	orm, err := postgres.NewClient(&postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      "integration",
		SSLMode: conf.PostgreSQL.SSLMode,
	}, logger)
	if err != nil {
		return fmt.Errorf("new postgres client: %w", err)
	}
	dbm := db.Database{ORM: orm}

	if err := IntegrationTypesMigration(conf, logger, dbm, m.AttachmentFolderPath()+"/integration_types.json"); err != nil {
		logger.Fatal("onboard migration failed", zap.Error(err))
		return err
	}

	parser := GitParser{}
	err = parser.ExtractConnectionGroups(m.AttachmentFolderPath())
	if err != nil {
		return err
	}

	err = dbm.ORM.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&integrationModels.IntegrationGroup{}).Where("1 = 1").Unscoped().Delete(&integrationModels.IntegrationGroup{}).Error
		if err != nil {
			logger.Error("failed to delete integration groups", zap.Error(err))
			return err
		}

		for _, integrationGroup := range parser.integrationGroups {
			err = tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&integrationGroup).Error
			if err != nil {
				logger.Error("failed to create integration group", zap.Error(err))
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failure in integration group transaction: %w", err)
	}

	return nil
}

func IntegrationTypesMigration(conf config.MigratorConfig, logger *zap.Logger, dbm db.Database, onboardFilePath string) error {
	content, err := os.ReadFile(onboardFilePath)
	if err != nil {
		return err
	}

	logger.Info("connectors json:", zap.String("json", string(content)))

	var integrationTypes []integrationModels.IntegrationType
	err = json.Unmarshal(content, &integrationTypes)
	if err != nil {
		return err
	}

	for _, obj := range integrationTypes {
		logger.Info("integrationType", zap.Any("obj", obj))
		err := dbm.ORM.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "name"}}, // key colume
			DoUpdates: clause.AssignmentColumns([]string{"id", "label", "short_description", "description",
				"enabled", "logo", "labels", "annotations", "tier"}),
		}).Create(&obj).Error
		if err != nil {
			return err
		}
	}

	return nil
}
