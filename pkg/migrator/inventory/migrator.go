package inventory

import (
	"encoding/json"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/db"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
	"os"
	"strings"
)

type ResourceType struct {
	ResourceName         string
	ResourceLabel        string
	ServiceName          string
	ListDescriber        string
	GetDescriber         string
	TerraformName        []string
	TerraformNameString  string `json:"-"`
	TerraformServiceName string
	FastDiscovery        bool
}

func Run(db db.Database, logger *zap.Logger, folder string) error {
	awsResourceTypesContent, err := os.ReadFile(folder + "aws-resource-types.json")
	if err != nil {
		return err
	}
	azureResourceTypesContent, err := os.ReadFile(folder + "azure-resource-types.json")
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

	for _, resourceType := range awsResourceTypes {
		err = db.ORM.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&inventory.Service{
			ServiceName:  strings.ToLower(resourceType.ServiceName),
			ServiceLabel: resourceType.ServiceName,
			Connector:    source.CloudAWS,
		}).Error
		if err != nil {
			return err
		}
		err = db.ORM.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&inventory.ResourceType{
			Connector:     source.CloudAWS,
			ResourceType:  resourceType.ResourceName,
			ResourceLabel: resourceType.ResourceLabel,
			ServiceName:   strings.ToLower(resourceType.ServiceName),
		}).Error
		if err != nil {
			return err
		}
	}

	for _, resourceType := range azureResourceTypes {
		err = db.ORM.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&inventory.Service{
			ServiceName:  strings.ToLower(resourceType.ServiceName),
			ServiceLabel: resourceType.ServiceName,
			Connector:    source.CloudAzure,
		}).Error
		if err != nil {
			return err
		}
		err = db.ORM.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&inventory.ResourceType{
			Connector:     source.CloudAzure,
			ResourceType:  resourceType.ResourceName,
			ResourceLabel: resourceType.ResourceLabel,
			ServiceName:   strings.ToLower(resourceType.ServiceName),
		}).Error
		if err != nil {
			return err
		}
	}

	return nil
}
