package db

func (s *Database) FindRDSInstancePrice(regionCode string, instanceType string, databaseEngine string, databaseEdition string,
	licenseModel string, deploymentOption string, costUnit string) (*AwsRdsInstancePrice, error) {
	var dbInstance AwsRdsInstancePrice
	tx := s.orm.Model(&AwsRdsInstancePrice{}).Where("region_code = ?", regionCode).Where("instance_type = ?", instanceType).
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

func (s *Database) FindRDSDBStoragePrice(regionCode string, deploymentOption string, volumeType string, costUnit string) (*AwsRdsStoragePrice, error) {
	var dbStorage AwsRdsStoragePrice
	tx := s.orm.Model(AwsRdsStoragePrice{}).Where("deployment_option = ?", deploymentOption).
		Where("region_code = ?", regionCode).Where("price_unit = ?", costUnit).
		Where("volume_type = ?", volumeType).Find(&dbStorage)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &dbStorage, nil
}

func (s *Database) FindRDSDBIopsPrice(regionCode string, deploymentOption string, costUnit string) (*AwsRdsIopsPrice, error) {
	var DBIops AwsRdsIopsPrice
	tx := s.orm.Model(AwsRdsIopsPrice{}).Where("deployment_option = ?", deploymentOption).
		Where("region_code = ?", regionCode).Where("price_unit = ?", costUnit).Find(&DBIops)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &DBIops, nil
}

func (s *Database) FindEC2InstancePrice(regionCode string, capacityStatus string, instanceType string, tenancy string,
	operatingSystem string, preInstalledSW string, costUnit string) (*AwsEC2InstancePrice, error) {
	var instance AwsEC2InstancePrice
	err := s.orm.Model(&AwsEC2InstancePrice{}).Where("region_code = ?", regionCode).
		Where("capacity_status = ?", capacityStatus).Where("instance_type = ?", instanceType).Where("tenancy = ?", tenancy).
		Where("operating_system = ?", operatingSystem).Where("pre_installed_sw = ?", preInstalledSW).
		Where("price_unit = ?", costUnit).Find(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (s *Database) FindEbsOptimizedPrice(regionCode string, instanceType string, usageType string, costUnit string) (*AwsEC2InstancePrice, error) {
	var instance AwsEC2InstancePrice
	err := s.orm.Model(&AwsEC2InstancePrice{}).Where("region_code = ?", regionCode).
		Where("instance_type = ?", instanceType).Where("usage_type = ?", usageType).
		Where("price_unit = ?", costUnit).Find(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (s *Database) FindEC2InstanceSystemOperationPrice(regionCode string, VolumeAPIName string, UsageType string) (*AwsEC2InstanceSystemOperationPrice, error) {
	var systemOperation AwsEC2InstanceSystemOperationPrice
	err := s.orm.Model(&AwsEC2InstanceSystemOperationPrice{}).Where("region_code = ?", regionCode).
		Where("volume_api_name = ?", VolumeAPIName).
		Where("usage_type LIKE '%?%'", UsageType).Find(&systemOperation).Error
	if err != nil {
		return nil, err
	}
	return &systemOperation, nil
}

func (s *Database) FindEC2InstanceStoragePrice(regionCode string, VolumeAPIName string) (*AwsEC2InstanceStoragePrice, error) {
	var storage AwsEC2InstanceStoragePrice
	err := s.orm.Model(&AwsEC2InstanceStoragePrice{}).Where("region_code = ?", regionCode).
		Where("volume_api_name = ?", VolumeAPIName).Find(&storage).Error
	if err != nil {
		return nil, err
	}
	return &storage, nil
}

func (s *Database) FindAmazonCloudWatchPrice(regionCode string, BeginRange int, costUnit string) (*AwsCloudwatchPrice, error) {
	var cloudWatch AwsCloudwatchPrice
	err := s.orm.Model(&AwsCloudwatchPrice{}).Where("region_code = ?", regionCode).Where("price_unit = ?", costUnit).
		Where("begin_range = ?", BeginRange).Find(&cloudWatch).Error
	if err != nil {
		return nil, err
	}
	return &cloudWatch, nil
}

func (s *Database) FindEC2CpuCreditsPrice(regionCode string, operatingSystem string, usageType string, costUnit string) (*AwsEC2CpuCreditsPrice, error) {
	var cpuCredits AwsEC2CpuCreditsPrice
	err := s.orm.Model(&AwsEC2CpuCreditsPrice{}).Where("region_code = ?", regionCode).
		Where("operating_system = ?", operatingSystem).Where("usageType = ?", usageType).Where("price_unit = ?", costUnit).Find(&cpuCredits).Error
	if err != nil {
		return nil, err
	}
	return &cpuCredits, nil
}

func (s *Database) FindLBPrice(regionCode string, productFamily string, usageType string, costUnit string) (*AwsElasticLoadBalancingPrice, error) {
	var lb AwsElasticLoadBalancingPrice
	err := s.orm.Model(&AwsElasticLoadBalancingPrice{}).Where("region_code = ?", regionCode).
		Where("product_family = ?", productFamily).Where("usage_type LIKE '%?%'", usageType).Where("price_unit = ?", costUnit).Find(&lb).Error
	if err != nil {
		return nil, err
	}
	return &lb, nil
}

func (s *Database) FindAzureVMPrice(regionCode string, size string, priority string) ([]*AzureVirtualMachinePrice, error) {
	var vm []*AzureVirtualMachinePrice
	err := s.orm.Model(&AzureVirtualMachinePrice{}).Where("arm_region_name = ?", regionCode).
		Where("arm_sku_name = ?", size).Where("priority = ?", priority).
		Find(&vm).Error
	if err != nil {
		return nil, err
	}
	return vm, nil
}

func (s *Database) FindAzureManagedStoragePrice(regionCode string, skuName string, meter string) (*AzureManagedStoragePrice, error) {
	var ms *AzureManagedStoragePrice
	err := s.orm.Model(&AzureManagedStoragePrice{}).Where("arm_region_name = ?", regionCode).
		Where("sku_name = ?", skuName).Where("meter = ?", meter).Find(&ms).Error
	if err != nil {
		return nil, err
	}
	return ms, nil
}

func (s *Database) FindAzureLoadBalancerPrice(regionCode string, meterName string) (*AzureLoadBalancerPrice, error) {
	var lb *AzureLoadBalancerPrice
	err := s.orm.Model(&AzureLoadBalancerPrice{}).Where("arm_region_name = ?", regionCode).
		Where("meter_name = ?", meterName).Find(&lb).Error
	if err != nil {
		return nil, err
	}
	return lb, nil
}
