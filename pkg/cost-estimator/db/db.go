package db

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/gorm"
)

type Database struct {
	orm *gorm.DB
}

func NewDatabase(orm *gorm.DB) Database {
	return Database{orm: orm}
}

func (db Database) Initialize() error {
	err := db.orm.AutoMigrate(
		&StoreCostTableJob{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (db Database) CreateStoreCostTableJob(connector source.Type) (uint, error) {
	job := StoreCostTableJob{Connector: connector, Status: StoreCostTableJobStatusProcessing}
	err := db.orm.Model(&StoreCostTableJob{}).Create(&job).Error
	if err != nil {
		return 0, err
	}
	return job.Id, nil
}

func (db Database) UpdateStoreCostTableJob(id uint, status StoreCostTableJobStatus, errorMessage string, count int64) error {
	return db.orm.Model(&StoreCostTableJob{}).Where("id = ?", id).
		Updates(StoreCostTableJob{Status: status, ErrorMessage: errorMessage, Count: count}).Error
}

func (db Database) GetLastJob(connector source.Type) (StoreCostTableJob, error) {
	var job StoreCostTableJob
	err := db.orm.Model(&StoreCostTableJob{}).Where("connector = ?", connector).Order("UpdatedAt").First(&job).Error
	if err != nil {
		return StoreCostTableJob{}, err
	}
	return job, nil
}

func (db Database) FindEC2InstanceCost(regionCode string, capacityStatus string, instanceType string, tenancy string,
	operatingSystem string, preInstalledSW string, costUnit string) (*EC2InstanceCost, error) {
	var instance EC2InstanceCost
	err := db.orm.Model(&EC2InstanceCost{}).Where("regionCode = ?", regionCode).
		Where("capacityStatus = ?", capacityStatus).Where("instanceType = ?", instanceType).Where("tenancy = ?", tenancy).
		Where("operatingSystem = ?", operatingSystem).Where("preInstalledSW = ?", preInstalledSW).
		Where("costUnit = ?", costUnit).Find(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (db Database) FindEbsOptimizedCost(regionCode string, instanceType string, usageType string, costUnit string) (*EC2InstanceCost, error) {
	var instance EC2InstanceCost
	err := db.orm.Model(&EC2InstanceCost{}).Where("regionCode = ?", regionCode).
		Where("instanceType = ?", instanceType).Where("UsageType = ?", usageType).
		Where("costUnit = ?", costUnit).Find(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (db Database) FindEC2InstanceSystemOperationCost(regionCode string, VolumeAPIName string, UsageType string) (*EC2InstanceSystemOperationCost, error) {
	var systemOperation EC2InstanceSystemOperationCost
	err := db.orm.Model(&EC2InstanceSystemOperationCost{}).Where("regionCode = ?", regionCode).
		Where("VolumeAPIName = ?", VolumeAPIName).
		Where("UsageType LIKE '%?%'", UsageType).Find(&systemOperation).Error
	if err != nil {
		return nil, err
	}
	return &systemOperation, nil
}

func (db Database) FindEC2InstanceStorageCost(regionCode string, VolumeAPIName string) (*EC2InstanceStorageCost, error) {
	var storage EC2InstanceStorageCost
	err := db.orm.Model(&EC2InstanceStorageCost{}).Where("regionCode = ?", regionCode).
		Where("VolumeAPIName = ?", VolumeAPIName).Find(&storage).Error
	if err != nil {
		return nil, err
	}
	return &storage, nil
}

func (db Database) FindAmazonCloudWatchCost(regionCode string, StartingRange int, costUnit string) (*AmazonCloudWatchCost, error) {
	var cloudWatch AmazonCloudWatchCost
	err := db.orm.Model(&AmazonCloudWatchCost{}).Where("regionCode = ?", regionCode).Where("costUnit = ?", costUnit).
		Where("StartingRange = ?", StartingRange).Find(&cloudWatch).Error
	if err != nil {
		return nil, err
	}
	return &cloudWatch, nil
}

func (db Database) FindEC2CpuCreditsCost(regionCode string, operatingSystem string, usageType string, costUnit string) (*EC2CpuCreditsCost, error) {
	var cpuCredits EC2CpuCreditsCost
	err := db.orm.Model(&EC2CpuCreditsCost{}).Where("regionCode = ?", regionCode).
		Where("operatingSystem = ?", operatingSystem).Where("usageType = ?", usageType).Where("costUnit = ?", costUnit).Find(&cpuCredits).Error
	if err != nil {
		return nil, err
	}
	return &cpuCredits, nil
}
