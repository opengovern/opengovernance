package inventory

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/db"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func Run(db db.Database, logger *zap.Logger, folder string) error {
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

	err = db.ORM.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&inventory.ResourceTypeTag{}).Joins("JOIN resource_types ON resource_types.resource_type = resource_type_tags.resource_type").Where("resource_types.connector = ?", source.CloudAWS).Delete(&inventory.ResourceTypeTag{}).Error
		if err != nil {
			logger.Error("failed to delete aws resource type tags", zap.Error(err))
			return err
		}
		err = tx.Model(&inventory.ResourceType{}).Where("connector = ?", source.CloudAWS).Delete(&inventory.ResourceType{}).Error
		if err != nil {
			logger.Error("failed to delete aws resource types", zap.Error(err))
			return err
		}
		err = tx.Model(&inventory.Service{}).Where("connector = ?", source.CloudAWS).Delete(&inventory.Service{}).Error
		if err != nil {
			logger.Error("failed to delete aws services", zap.Error(err))
			return err
		}

		for _, resourceType := range awsResourceTypes {
			err = tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&inventory.Service{
				ServiceName:  strings.ToLower(resourceType.ServiceName),
				ServiceLabel: resourceType.ServiceName,
				Connector:    source.CloudAWS,
			}).Error
			if err != nil {
				logger.Error("failed to create aws service", zap.Error(err))
				return err
			}
			err = tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&inventory.ResourceType{
				Connector:     source.CloudAWS,
				ResourceType:  resourceType.ResourceName,
				ResourceLabel: resourceType.ResourceLabel,
				ServiceName:   strings.ToLower(resourceType.ServiceName),
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

	err = db.ORM.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&inventory.ResourceTypeTag{}).Joins("JOIN resource_types ON resource_types.resource_type = resource_type_tags.resource_type").Where("resource_types.connector = ?", source.CloudAzure).Delete(&inventory.ResourceTypeTag{}).Error
		if err != nil {
			logger.Error("failed to delete azure resource type tags", zap.Error(err))
			return err
		}
		err = tx.Model(&inventory.ResourceType{}).Where("connector = ?", source.CloudAzure).Delete(&inventory.ResourceType{}).Error
		if err != nil {
			logger.Error("failed to delete azure resource types", zap.Error(err))
			return err
		}
		err = tx.Model(&inventory.Service{}).Where("connector = ?", source.CloudAzure).Delete(&inventory.Service{}).Error
		if err != nil {
			logger.Error("failed to delete azure services", zap.Error(err))
			return err
		}

		for _, resourceType := range azureResourceTypes {
			err = tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&inventory.Service{
				ServiceName:  strings.ToLower(resourceType.ServiceName),
				ServiceLabel: resourceType.ServiceName,
				Connector:    source.CloudAzure,
			}).Error
			if err != nil {
				logger.Error("failed to create azure service", zap.Error(err))
				return err
			}
			err = tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&inventory.ResourceType{
				Connector:     source.CloudAzure,
				ResourceType:  resourceType.ResourceName,
				ResourceLabel: resourceType.ResourceLabel,
				ServiceName:   strings.ToLower(resourceType.ServiceName),
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
