package db

import (
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CostEstimatorDatabase struct {
	orm *gorm.DB
}

func NewCostEstimatorDatabase(settings *postgres.Config, logger *zap.Logger) (*CostEstimatorDatabase, error) {
	orm, err := postgres.NewClient(settings, logger)
	if err != nil {
		return nil, fmt.Errorf("new cost estimator postgres client: %w", err)
	}
	return &CostEstimatorDatabase{orm: orm}, nil
}

func (db CostEstimatorDatabase) FindRDSInstancePrice(regionCode string, instanceType string, databaseEngine string, databaseEdition string,
	licenseModel string, deploymentOption string, costUnit string) (*AwsRdsInstancePrice, error) {
	var dbInstance AwsRdsInstancePrice
	tx := db.orm.Model(&AwsRdsInstancePrice{}).Where("region_code = ?", regionCode).Where("instance_type = ?", instanceType).
		Where("database_engine = ?", databaseEngine).
		Where("deployment_option = ?", deploymentOption).Where("price_unit = ?", costUnit).Find(&dbInstance)
	if databaseEngine != "" {
		tx = tx.Where("database_edition = ?", databaseEdition)
	}
	if licenseModel != "" {
		tx = tx.Where("license_model = ?", licenseModel)
	}
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &dbInstance, nil
}

func (db CostEstimatorDatabase) FindRDSDBStoragePrice(regionCode string, deploymentOption string, volumeType string, costUnit string) (*AwsRdsStoragePrice, error) {
	var dbStorage AwsRdsStoragePrice
	tx := db.orm.Model(AwsRdsStoragePrice{}).Where("deployment_option = ?", deploymentOption).
		Where("region_code = ?", regionCode).Where("price_unit = ?", costUnit).
		Where("volume_type = ?", volumeType).Find(&dbStorage)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &dbStorage, nil
}

func (db CostEstimatorDatabase) FindRDSDBIopsPrice(regionCode string, deploymentOption string, costUnit string) (*AwsRdsIopsPrice, error) {
	var DBIops AwsRdsIopsPrice
	tx := db.orm.Model(AwsRdsIopsPrice{}).Where("deployment_option = ?", deploymentOption).
		Where("region_code = ?", regionCode).Where("price_unit = ?", costUnit).Find(&DBIops)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &DBIops, nil
}

func (db CostEstimatorDatabase) FindEC2InstancePrice(regionCode string, capacityStatus string, instanceType string, tenancy string,
	operatingSystem string, preInstalledSW string, costUnit string) (*AwsEC2InstancePrice, error) {
	var instance AwsEC2InstancePrice
	err := db.orm.Model(&AwsEC2InstancePrice{}).Where("region_code = ?", regionCode).
		Where("capacity_status = ?", capacityStatus).Where("instance_type = ?", instanceType).Where("tenancy = ?", tenancy).
		Where("operating_system = ?", operatingSystem).Where("pre_installed_sw = ?", preInstalledSW).
		Where("price_unit = ?", costUnit).Find(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (db CostEstimatorDatabase) FindEbsOptimizedPrice(regionCode string, instanceType string, usageType string, costUnit string) (*AwsEC2InstancePrice, error) {
	var instance AwsEC2InstancePrice
	err := db.orm.Model(&AwsEC2InstancePrice{}).Where("region_code = ?", regionCode).
		Where("instance_type = ?", instanceType).Where("usage_type = ?", usageType).
		Where("price_unit = ?", costUnit).Find(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (db CostEstimatorDatabase) FindEC2InstanceSystemOperationPrice(regionCode string, VolumeAPIName string, UsageType string) (*AwsEC2InstanceSystemOperationPrice, error) {
	var systemOperation AwsEC2InstanceSystemOperationPrice
	err := db.orm.Model(&AwsEC2InstanceSystemOperationPrice{}).Where("region_code = ?", regionCode).
		Where("volume_api_name = ?", VolumeAPIName).
		Where("usage_type LIKE '%?%'", UsageType).Find(&systemOperation).Error
	if err != nil {
		return nil, err
	}
	return &systemOperation, nil
}

func (db CostEstimatorDatabase) FindEC2InstanceStoragePrice(regionCode string, VolumeAPIName string) (*AwsEC2InstanceStoragePrice, error) {
	var storage AwsEC2InstanceStoragePrice
	err := db.orm.Model(&AwsEC2InstanceStoragePrice{}).Where("region_code = ?", regionCode).
		Where("volume_api_name = ?", VolumeAPIName).Find(&storage).Error
	if err != nil {
		return nil, err
	}
	return &storage, nil
}

func (db CostEstimatorDatabase) FindAmazonCloudWatchPrice(regionCode string, BeginRange int, costUnit string) (*AwsCloudwatchPrice, error) {
	var cloudWatch AwsCloudwatchPrice
	err := db.orm.Model(&AwsCloudwatchPrice{}).Where("region_code = ?", regionCode).Where("price_unit = ?", costUnit).
		Where("begin_range = ?", BeginRange).Find(&cloudWatch).Error
	if err != nil {
		return nil, err
	}
	return &cloudWatch, nil
}

func (db CostEstimatorDatabase) FindEC2CpuCreditsPrice(regionCode string, operatingSystem string, usageType string, costUnit string) (*AwsEC2CpuCreditsPrice, error) {
	var cpuCredits AwsEC2CpuCreditsPrice
	err := db.orm.Model(&AwsEC2CpuCreditsPrice{}).Where("region_code = ?", regionCode).
		Where("operating_system = ?", operatingSystem).Where("usageType = ?", usageType).Where("price_unit = ?", costUnit).Find(&cpuCredits).Error
	if err != nil {
		return nil, err
	}
	return &cpuCredits, nil
}

func (db CostEstimatorDatabase) FindLBPrice(regionCode string, productFamily string, usageType string, costUnit string) (*AwsElasticLoadBalancingPrice, error) {
	var lb AwsElasticLoadBalancingPrice
	err := db.orm.Model(&AwsElasticLoadBalancingPrice{}).Where("region_code = ?", regionCode).
		Where("product_family = ?", productFamily).Where("usage_type LIKE '%?%'", usageType).Where("price_unit = ?", costUnit).Find(&lb).Error
	if err != nil {
		return nil, err
	}
	return &lb, nil
}

func (db CostEstimatorDatabase) FindAzureVMPrice(regionCode string, size string, priority string) ([]*AzureVirtualMachinePrice, error) {
	var vm []*AzureVirtualMachinePrice
	err := db.orm.Model(&AzureVirtualMachinePrice{}).Where("arm_region_name = ?", regionCode).
		Where("arm_sku_name = ?", size).Where("priority = ?", priority).
		Find(&vm).Error
	if err != nil {
		return nil, err
	}
	return vm, nil
}
