package inventory

import (
	"encoding/json"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/db"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"os"
	"path"
	"strings"
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

func RunResourceType(conf postgres.Config, logger *zap.Logger, folder string) error {
	conf.DB = "inventory"
	orm, err := postgres.NewClient(&conf, logger)
	if err != nil {
		logger.Error("failed to create postgres client", zap.Error(err))
		return fmt.Errorf("new postgres client: %w", err)
	}
	dbConn, err := orm.DB()
	if err != nil {
		logger.Error("failed to get db from orm", zap.Error(err))
		return err
	}
	defer dbConn.Close()

	dbm := db.Database{ORM: orm}

	awsResourceTypesContent, err := os.ReadFile(path.Join(folder, "aws-resource-types.json"))
	if err != nil {
		return err
	}
	azureResourceTypesContent, err := os.ReadFile(path.Join(folder, "azure-resource-types.json"))
	if err != nil {
		return err
	}
	var awsResourceTypes []ResourceType
	var azureResourceTypes []ResourceType
	if err := json.Unmarshal(awsResourceTypesContent, &awsResourceTypes); err != nil {
		return err
	}
	if err := json.Unmarshal(azureResourceTypesContent, &azureResourceTypes); err != nil {
		return err
	}

	err = dbm.ORM.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&inventory.ResourceType{}).Where("connector = ?", source.CloudAWS).Unscoped().Delete(&inventory.ResourceType{}).Error
		if err != nil {
			logger.Error("failed to delete aws resource types", zap.Error(err))
			return err
		}

		for _, resourceType := range awsResourceTypes {
			err = tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&inventory.ResourceType{
				Connector:     source.CloudAWS,
				ResourceType:  resourceType.ResourceName,
				ResourceLabel: resourceType.ResourceLabel,
				ServiceName:   strings.ToLower(resourceType.ServiceName),
				DoSummarize:   !resourceType.IgnoreSummarize,
			}).Error
			if err != nil {
				logger.Error("failed to create aws resource type", zap.Error(err))
				return err
			}

			err = tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&inventory.ResourceTypeTag{
				Tag: model.Tag{
					Key:   "category",
					Value: resourceType.Category,
				},
				ResourceType: resourceType.ResourceName,
			}).Error
			if err != nil {
				logger.Error("failed to create aws resource type tag", zap.Error(err))
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failure in aws transaction: %v", err)
	}

	err = dbm.ORM.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&inventory.ResourceType{}).Where("connector = ?", source.CloudAzure).Unscoped().Delete(&inventory.ResourceType{}).Error
		if err != nil {
			logger.Error("failed to delete azure resource types", zap.Error(err))
			return err
		}
		for _, resourceType := range azureResourceTypes {
			err = tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&inventory.ResourceType{
				Connector:     source.CloudAzure,
				ResourceType:  resourceType.ResourceName,
				ResourceLabel: resourceType.ResourceLabel,
				ServiceName:   strings.ToLower(resourceType.ServiceName),
				DoSummarize:   !resourceType.IgnoreSummarize,
			}).Error
			if err != nil {
				logger.Error("failed to create azure resource type", zap.Error(err))
				return err
			}

			err = tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&inventory.ResourceTypeTag{
				Tag: model.Tag{
					Key:   "category",
					Value: resourceType.Category,
				},
				ResourceType: resourceType.ResourceName,
			}).Error
			if err != nil {
				logger.Error("failed to create azure resource type tag", zap.Error(err))
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failure in azure transaction: %v", err)
	}

	return nil
}

type ResourceCollection struct {
	ID      string                           `json:"id"`
	Name    string                           `json:"name"`
	Tags    map[string][]string              `json:"tags"`
	Filters []kaytu.ResourceCollectionFilter `json:"filters"`
}

func RunResourceCollection(conf postgres.Config, logger *zap.Logger, directory string) error {
	conf.DB = "inventory"
	orm, err := postgres.NewClient(&conf, logger)
	if err != nil {
		logger.Error("failed to create postgres client", zap.Error(err))
		return fmt.Errorf("new postgres client: %w", err)
	}
	dbConn, err := orm.DB()
	if err != nil {
		logger.Error("failed to get db from orm", zap.Error(err))
		return err
	}
	defer dbConn.Close()

	dbm := db.Database{ORM: orm}

	resourceCollections, err := ExtractResourceCollections(directory)
	if err != nil {
		logger.Error("failed to extract resource collections", zap.Error(err))
		return err
	}

	err = dbm.ORM.Transaction(func(tx *gorm.DB) error {
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

			dbResourceCollection := inventory.ResourceCollection{
				ID:          resourceCollection.ID,
				Name:        resourceCollection.Name,
				FiltersJson: jsonb,
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
