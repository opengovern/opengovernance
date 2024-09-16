package resource_collection

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/open-governance/pkg/inventory"
	"github.com/kaytu-io/open-governance/services/migrator/config"
	"github.com/kaytu-io/open-governance/services/migrator/db"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type ResourceCollection struct {
	ID          string                             `json:"id" yaml:"id"`
	Name        string                             `json:"name" yaml:"name"`
	Tags        map[string][]string                `json:"tags" yaml:"tags"`
	Filters     []kaytu.ResourceCollectionFilter   `json:"filters" yaml:"filters"`
	Description string                             `json:"description" yaml:"description"`
	Status      inventory.ResourceCollectionStatus `json:"status" yaml:"status"`
}

type Migration struct {
}

func (m Migration) AttachmentFolderPath() string {
	return config.ResourceCollectionGitPath
}
func (m Migration) IsGitBased() bool {
	return true
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

	resourceCollections, err := ExtractResourceCollections(m.AttachmentFolderPath())
	if err != nil {
		logger.Error("failed to extract resource collections", zap.Error(err))
		return err
	}

	err = dbm.ORM.Transaction(func(tx *gorm.DB) error {
		currentRCs := make([]inventory.ResourceCollection, 0)
		err := tx.Model(&inventory.ResourceCollection{}).Find(&currentRCs).Error
		if err != nil {
			logger.Error("failed to get current resource collections", zap.Error(err))
			return err
		}
		currentRcMap := make(map[string]inventory.ResourceCollection)
		for _, rc := range currentRCs {
			currentRcMap[rc.ID] = rc
		}

		tx.Model(&inventory.ResourceCollection{}).Where("1=1").Unscoped().Delete(&inventory.ResourceCollection{})
		tx.Model(&inventory.ResourceCollectionTag{}).Where("1=1").Unscoped().Delete(&inventory.ResourceCollectionTag{})
		for _, resourceCollection := range resourceCollections {
			filtersJson, err := json.Marshal(resourceCollection.Filters)
			if err != nil {
				logger.Error("failed to marshal filters", zap.Error(err))
				return err
			}

			jsonb := pgtype.JSONB{}
			err = jsonb.Set(filtersJson)
			if err != nil {
				logger.Error("failed to set jsonb", zap.Error(err))
				return err
			}

			createdAt := time.Now()
			if currentRc, ok := currentRcMap[resourceCollection.ID]; ok {
				createdAt = currentRc.Created
				if createdAt.IsZero() || createdAt.Year() == 1 {
					createdAt = time.Now()
				}
			}
			if resourceCollection.Status == "" {
				resourceCollection.Status = inventory.ResourceCollectionStatusActive
			}

			dbResourceCollection := inventory.ResourceCollection{
				ID:          resourceCollection.ID,
				Name:        resourceCollection.Name,
				FiltersJson: jsonb,
				Description: resourceCollection.Description,
				Status:      resourceCollection.Status,
				Created:     createdAt,
			}
			err = tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&dbResourceCollection).Error
			if err != nil {
				logger.Error("failed to create resource collection", zap.Error(err))
				return err
			}

			for key, values := range resourceCollection.Tags {
				err = tx.Clauses(clause.OnConflict{
					DoNothing: true,
				}).Create(&inventory.ResourceCollectionTag{
					Tag: model.Tag{
						Key:   key,
						Value: values,
					},
					ResourceCollectionID: resourceCollection.ID,
				}).Error
				if err != nil {
					logger.Error("failed to create resource collection tag", zap.Error(err))
					return err
				}
			}
		}
		return nil
	})

	return nil
}
