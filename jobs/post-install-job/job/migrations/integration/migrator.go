package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jackc/pgtype"
	"github.com/opengovern/opencomply/jobs/post-install-job/config"
	"github.com/opengovern/opencomply/jobs/post-install-job/db"
	integration_type "github.com/opengovern/opencomply/services/integration/integration-type"
	integrationModels "github.com/opengovern/opencomply/services/integration/models"
	"gorm.io/gorm/clause"

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
	return config.IntegrationsGitPath
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

	if err := IntegrationTypesMigration(conf, logger, dbm, "/integrations/integration_types.json"); err != nil {
		logger.Fatal("integration migration failed", zap.Error(err))
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

type IntegrationType struct {
	ID               int64               `json:"id"`
	Name             string              `json:"name"`
	IntegrationType  string              `json:"integration_type"`
	Label            string              `json:"label"`
	Tier             string              `json:"tier"`
	Annotations      map[string][]string `json:"annotations"`
	Labels           map[string][]string `json:"labels"`
	ShortDescription string              `json:"short_description"`
	Description      string              `json:"description"`
	Logo             string              `json:"logo"`
	Enabled          bool                `json:"enabled"`
}

func IntegrationTypesMigration(conf config.MigratorConfig, logger *zap.Logger, dbm db.Database, onboardFilePath string) error {
	content, err := os.ReadFile(onboardFilePath)
	if err != nil {
		return err
	}

	logger.Info("integration types json:", zap.String("json", string(content)))

	var integrationTypes []IntegrationType
	err = json.Unmarshal(content, &integrationTypes)
	if err != nil {
		return err
	}

	err = dbm.ORM.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&integrationModels.IntegrationType{}).Where("1 = 1").Unscoped().Delete(&integrationModels.IntegrationType{}).Error
		if err != nil {
			logger.Error("failed to delete integration types", zap.Error(err))
			return err
		}

		for _, obj := range integrationTypes {
			integrationType := integrationModels.IntegrationType{
				ID:               obj.ID,
				IntegrationType:  obj.IntegrationType,
				Name:             obj.Name,
				Label:            obj.Label,
				Tier:             obj.Tier,
				ShortDescription: obj.ShortDescription,
				Description:      obj.Description,
				Logo:             obj.Logo,
				Enabled:          obj.Enabled,
			}
			if _, ok := integration_type.IntegrationTypes[integration_type.ParseType(integrationType.IntegrationType)]; ok {
				integrationType.Enabled = true
			} else {
				integrationType.Enabled = false
			}
			annotationsJsonData, err := json.Marshal(obj.Annotations)
			if err != nil {
				return err
			}
			integrationAnnotationsJsonb := pgtype.JSONB{}
			err = integrationAnnotationsJsonb.Set(annotationsJsonData)
			integrationType.Annotations = integrationAnnotationsJsonb

			labelsJsonData, err := json.Marshal(obj.Labels)
			if err != nil {
				return err
			}
			integrationLabelsJsonb := pgtype.JSONB{}
			err = integrationLabelsJsonb.Set(labelsJsonData)
			integrationType.Labels = integrationLabelsJsonb

			logger.Info("integrationType", zap.Any("obj", obj))
			err = tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&integrationType).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failure in integration types transaction: %w", err)
	}

	return nil
}
