package resource_info

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/opengovern/og-util/pkg/postgres"
	"github.com/opengovern/opengovernance/pkg/inventory"
	integration_type "github.com/opengovern/opengovernance/services/integration/integration-type"
	"github.com/opengovern/opengovernance/services/migrator/config"
	"github.com/opengovern/opengovernance/services/migrator/db"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"os"
	"path"
)

type ResourceType struct {
	ResourceName         string
	Category             []string
	ResourceLabel        string
	ServiceName          string
	ListDescriber        string
	GetDescriber         string
	TerraformName        []string
	TerraformNameString  string `json:"-"`
	TerraformServiceName string
	Discovery            string
	IgnoreSummarize      bool
	SteampipeTable       string
	Model                string
}

type Migration struct {
}

func (m Migration) IsGitBased() bool {
	return false
}

func (m Migration) AttachmentFolderPath() string {
	return "/inventory-data-config"
}

func (m Migration) Run(ctx context.Context, conf config.MigratorConfig, logger *zap.Logger) error {
	orm, err := postgres.NewClient(&postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      "inventory",
		SSLMode: conf.PostgreSQL.SSLMode,
	}, logger)
	if err != nil {
		return fmt.Errorf("new postgres client: %w", err)
	}
	dbm := db.Database{ORM: orm}

	awsFile, err := os.Open(path.Join(m.AttachmentFolderPath(), "aws-resource-types.csv"))
	if err != nil {
		return err
	}
	defer awsFile.Close()

	reader := csv.NewReader(awsFile)

	_, err = reader.Read()
	if err != nil {
		return err
	}

	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	err = dbm.ORM.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&inventory.ResourceTypeV2{}).Where("1 = 1").Unscoped().Delete(&inventory.ResourceTypeV2{}).Error
		if err != nil {
			logger.Error("failed to delete aws resource types", zap.Error(err))
			return err
		}

		for _, record := range records {
			err = tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&inventory.ResourceTypeV2{
				IntegrationType: integration_type.IntegrationTypeAWSAccount,
				ResourceName:    record[0],
				ResourceID:      record[1],
				SteampipeTable:  record[2],
				Category:        record[3],
			}).Error
			if err != nil {
				logger.Error("failed to create aws resource type", zap.Error(err))
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failure in aws transaction: %v", err)
	}

	return nil
}
