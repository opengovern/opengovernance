package db

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CostEstimatorDatabase struct {
	orm *gorm.DB
}

func NewCostEstimatorDatabase(settings *workspace.Config, logger *zap.Logger) (*CostEstimatorDatabase, error) {
	cfg := postgres.Config{
		Host:    settings.Host,
		Port:    settings.Port,
		User:    settings.User,
		Passwd:  settings.Password,
		DB:      settings.CostEstimatorDBName,
		SSLMode: settings.SSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new cost estimator postgres client: %w", err)
	}
	return &CostEstimatorDatabase{orm: orm}, nil
}

func (db CostEstimatorDatabase) FindEC2InstancePrice(regionCode string, capacityStatus string, instanceType string, tenancy string,
	operatingSystem string, preInstalledSW string, costUnit string) (*EC2InstancePrice, error) {
	var instance EC2InstancePrice
	err := db.orm.Model(&EC2InstancePrice{}).Where("regionCode = ?", regionCode).
		Where("capacityStatus = ?", capacityStatus).Where("instanceType = ?", instanceType).Where("tenancy = ?", tenancy).
		Where("operatingSystem = ?", operatingSystem).Where("preInstalledSW = ?", preInstalledSW).
		Where("costUnit = ?", costUnit).Find(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (db CostEstimatorDatabase) FindEbsOptimizedPrice(regionCode string, instanceType string, usageType string, costUnit string) (*EC2InstancePrice, error) {
	var instance EC2InstancePrice
	err := db.orm.Model(&EC2InstancePrice{}).Where("regionCode = ?", regionCode).
		Where("instanceType = ?", instanceType).Where("UsageType = ?", usageType).
		Where("costUnit = ?", costUnit).Find(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (db CostEstimatorDatabase) FindEC2InstanceSystemOperationPrice(regionCode string, VolumeAPIName string, UsageType string) (*EC2InstanceSystemOperationPrice, error) {
	var systemOperation EC2InstanceSystemOperationPrice
	err := db.orm.Model(&EC2InstanceSystemOperationPrice{}).Where("regionCode = ?", regionCode).
		Where("VolumeAPIName = ?", VolumeAPIName).
		Where("UsageType LIKE '%?%'", UsageType).Find(&systemOperation).Error
	if err != nil {
		return nil, err
	}
	return &systemOperation, nil
}

func (db CostEstimatorDatabase) FindEC2InstanceStoragePrice(regionCode string, VolumeAPIName string) (*EC2InstanceStoragePrice, error) {
	var storage EC2InstanceStoragePrice
	err := db.orm.Model(&EC2InstanceStoragePrice{}).Where("regionCode = ?", regionCode).
		Where("VolumeAPIName = ?", VolumeAPIName).Find(&storage).Error
	if err != nil {
		return nil, err
	}
	return &storage, nil
}

func (db CostEstimatorDatabase) FindAmazonCloudWatchPrice(regionCode string, StartingRange int, costUnit string) (*AmazonCloudWatchPrice, error) {
	var cloudWatch AmazonCloudWatchPrice
	err := db.orm.Model(&AmazonCloudWatchPrice{}).Where("regionCode = ?", regionCode).Where("costUnit = ?", costUnit).
		Where("StartingRange = ?", StartingRange).Find(&cloudWatch).Error
	if err != nil {
		return nil, err
	}
	return &cloudWatch, nil
}

func (db CostEstimatorDatabase) FindEC2CpuCreditsPrice(regionCode string, operatingSystem string, usageType string, costUnit string) (*EC2CpuCreditsCost, error) {
	var cpuCredits EC2CpuCreditsCost
	err := db.orm.Model(&EC2CpuCreditsCost{}).Where("regionCode = ?", regionCode).
		Where("operatingSystem = ?", operatingSystem).Where("usageType = ?", usageType).Where("costUnit = ?", costUnit).Find(&cpuCredits).Error
	if err != nil {
		return nil, err
	}
	return &cpuCredits, nil
}
