package azure

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"strings"
)

const (
	sqlServerlessTier = "general purpose - serverless"
	sqlHyperscaleTier = "hyperscale"
)

var (
	mssqlTierMapping = map[string]string{
		"b": "Basic",
		"p": "Premium",
		"s": "Standard",
	}

	mssqlStandardDTUIncludedStorage = map[string]float64{
		"P1":  500,
		"P2":  500,
		"P4":  500,
		"P6":  500,
		"P11": 4096,
		"P15": 4096,
	}

	mssqlStorageRedundancyTypeMapping = map[string]string{
		"geo":   "RA-GRS",
		"local": "LRS",
		"zone":  "ZRS",
	}
)

func SqlServerDatabaseCostByResource(db *db.Database, request api.GetAzureSqlServersDatabasesRequest) (float64, error) {
	if strings.ToLower(*request.SqlServerDB.Database.SKU.Name) == "elasticpool" {
		cost, err := elasticPoolCostComponents(db, request)
		if err != nil {
			return 0, err
		}
		return cost, nil
	}

	if request.SqlServerDB.Database.SKU.Capacity != nil {
		vCoreCost, err := vCoreCostComponents(db, request)
		if err != nil {
			return 0, err
		}
		return vCoreCost, nil
	}

	cost, err := dtuCostComponents(db, request)
	if err != nil {
		return 0, err
	}

	return cost, nil
}

func elasticPoolCostComponents(db *db.Database, request api.GetAzureSqlServersDatabasesRequest) (float64, error) {

	var cost float64

	longTermRetentionCost, err := longTermRetentionCostComponent(db, request)
	if err != nil {
		return 0, err
	}
	cost += longTermRetentionCost

	pitrBackupCost, err := pitrBackupCostComponent(db, request)
	if err != nil {
		return 0, err
	}
	cost += pitrBackupCost

	return cost, nil
}

func vCoreCostComponents(db *db.Database, request api.GetAzureSqlServersDatabasesRequest) (float64, error) {
	costComponents, err := computeHoursCostComponents(db, request)
	if err != nil {
		return 0, err
	}
	sku := request.SqlServerDB.Database.SKU

	if strings.ToLower(*sku.Tier) == sqlHyperscaleTier {
		readReplicaCost, err := readReplicaCostComponent(db, request)
		if err != nil {
			return 0, err
		}
		costComponents += readReplicaCost
	}
	if strings.ToLower(*sku.Tier) != sqlServerlessTier && strings.ToLower(string(*request.SqlServerDB.Database.Properties.LicenseType)) == "licenseincluded" {
		productName := fmt.Sprintf("/%s - %s/", *request.SqlServerDB.Database.SKU.Tier, "SQL License")
		response, err := db.FindAzureSqlServerDatabaseLicenseCostComponentPrice(request.RegionCode, "SQL Database", "Databases", productName, "vCore-hours")
		if err != nil {
			return 0, err
		}
		costComponents += response.Price
	}

	// check the max size field that is with byte type in our resource
	//	if maxSizeGB != nil {
	//		storageGB = decimalPtr(decimal.NewFromFloat(*maxSizeGB))
	//	}

	storageTier := *request.SqlServerDB.Database.SKU.Tier
	if strings.ToLower(storageTier) == "general purpose - serverless" {
		storageTier = "General Purpose"
	}
	skuName := storageTier
	if *request.SqlServerDB.Database.Properties.ZoneRedundant {
		skuName += " Zone Redundancy"
	}

	productNameRegex := fmt.Sprintf("/%s - Storage/", storageTier)
	StorageCostComponent, err := db.FindAzureSqlServerDatabaseLongTermRetentionPrice(request.RegionCode, skuName, productNameRegex, "Data Stored", "GB")
	if err != nil {
		return 0, err
	}
	costComponents += StorageCostComponent.Price

	if strings.ToLower(*request.SqlServerDB.Database.SKU.Tier) != sqlHyperscaleTier {
		longTermRetentionCost, err := longTermRetentionCostComponent(db, request)
		if err != nil {
			return 0, err
		}
		costComponents += longTermRetentionCost

		pitrBackupCost, err := pitrBackupCostComponent(db, request)
		if err != nil {
			return 0, err
		}
		costComponents += pitrBackupCost
	}
	return costComponents, nil
}

func dtuCostComponents(db *db.Database, request api.GetAzureSqlServersDatabasesRequest) (float64, error) {
	var cost float64
	skuName := strings.ToLower(*request.SqlServerDB.Database.SKU.Name)
	if skuName == "basic" {
		skuName = "b"
	}

	// we have problem here :
	productName := fmt.Sprintf("SQL Database Single %s", mssqlTierMapping[])
	meterName := fmt.Sprintln("DTUs")
	response, err := db.FindAzureSqlServerDatabaseLongTermRetentionPrice(request.RegionCode, skuName, productName, meterName, "hours")
	if err != nil {
		return 0, err
	}
	cost += response.Price

	// we need to check ExtraStorageGB to see if we have that field
	// actually it should implement right here

	longTermRetentionCost, err := longTermRetentionCostComponent(db, request)
	if err != nil {
		return 0, err
	}
	cost += longTermRetentionCost

	pitrBackupCost, err := pitrBackupCostComponent(db, request)
	if err != nil {
		return 0, err
	}
	cost += pitrBackupCost

	return cost, nil
}

func longTermRetentionCostComponent(dbFunc *db.Database, request api.GetAzureSqlServersDatabasesRequest) (float64, error) {
	// mssqlStorageRedundancyTypeMapping should assign the GeoZone type
	redundancyType, ok := mssqlStorageRedundancyTypeMapping[strings.ToLower(string(*request.SqlServerDB.Database.Properties.CurrentBackupStorageRedundancy))]
	if !ok {
		redundancyType = "RA-GRS"
	}

	skuName := fmt.Sprintf("Backup %s", redundancyType)
	productName := fmt.Sprintln("SQL Database - LTR Backup Storage")
	meterName := fmt.Sprintf("%s Data Stored", skuName)

	longTermRetentionCost, err := dbFunc.FindAzureSqlServerDatabaseLongTermRetentionPrice(request.RegionCode, skuName, productName, meterName, "GB")
	if err != nil {
		return 0, err
	}
	return longTermRetentionCost.Price, nil
}

func pitrBackupCostComponent(dbFunc *db.Database, request api.GetAzureSqlServersDatabasesRequest) (float64, error) {
	// mssqlStorageRedundancyTypeMapping should assign the GeoZone type
	redundancyType, ok := mssqlStorageRedundancyTypeMapping[strings.ToLower(string(*request.SqlServerDB.Database.Properties.CurrentBackupStorageRedundancy))]
	if !ok {
		redundancyType = "RA-GRS"
	}

	productName := fmt.Sprintln("SQL Database Single/Elastic Pool PITR Backup Storage")
	skuName := fmt.Sprintf("Backup %s", redundancyType)
	meterName := fmt.Sprintf("%s Data Stored", redundancyType)

	longTermRetentionCost, err := dbFunc.FindAzureSqlServerDatabaseLongTermRetentionPrice(request.RegionCode, skuName, productName, meterName, "GB")
	if err != nil {
		return 0, err
	}
	return longTermRetentionCost.Price, nil
}

func computeHoursCostComponents(db *db.Database, request api.GetAzureSqlServersDatabasesRequest) (float64, error) {
	if strings.ToLower(*request.SqlServerDB.Database.SKU.Tier) == sqlServerlessTier {
		responseCost, err := serverlessComputeHoursCostComponents(db, request)
		if err != nil {
			return 0, err
		}
		return responseCost, nil
	}

	responseCost, err := provisionedComputeCostComponents(db, request)
	if err != nil {
		return 0, err
	}
	return responseCost, nil
}

func serverlessComputeHoursCostComponents(dbfunc *db.Database, request api.GetAzureSqlServersDatabasesRequest) (float64, error) {
	var cost float64
	productName := fmt.Sprintf("%s - %s", *request.SqlServerDB.Database.SKU.Tier, *request.SqlServerDB.Database.SKU.Family)
	// check the meter name
	response, err := dbfunc.FindAzureSqlServerDatabaseLongTermRetentionPrice(request.RegionCode, "1 vCore", productName, "1 vCore - Free", "vCore-hours")
	if err != nil {
		return 0, err
	}
	cost += response.Price

	if *request.SqlServerDB.Database.Properties.ZoneRedundant {
		// we don't have any '1 vCore Zone Redundancy' sku name
		responseZoneRedundant, err := dbfunc.FindAzureSqlServerDatabaseLongTermRetentionPrice(request.RegionCode, "1 vCore Zone Redundancy", productName, "Data Stored", "vCore-hours")
		if err != nil {
			return 0, err
		}
		cost += responseZoneRedundant.Price
	}

	return cost, nil
}

func provisionedComputeCostComponents(dbfunc *db.Database, request api.GetAzureSqlServersDatabasesRequest) (float64, error) {
	var cost float64

	productName := fmt.Sprintf("%s - %s", *request.SqlServerDB.Database.SKU.Tier, *request.SqlServerDB.Database.SKU.Family)
	responseCost, err := dbfunc.FindAzureSqlServerDatabaseReadReplicaCostComponentPrice(request.RegionCode, fmt.Sprintf("%d vCore", *request.SqlServerDB.Database.SKU.Capacity), productName, "hours")
	if err != nil {
		return 0, err
	}
	cost += responseCost.Price

	if *request.SqlServerDB.Database.Properties.ZoneRedundant {
		ReadReplicaResponseCost, err := dbfunc.FindAzureSqlServerDatabaseReadReplicaCostComponentPrice(request.RegionCode, fmt.Sprintf("%d vCore Zone Redundancy", *request.SqlServerDB.Database.SKU.Capacity), productName, "hours")
		if err != nil {
			return 0, err
		}
		cost += ReadReplicaResponseCost.Price
	}

	return cost, nil
}

func readReplicaCostComponent(db *db.Database, request api.GetAzureSqlServersDatabasesRequest) (float64, error) {
	productName := fmt.Sprintf("%s - %s", *request.SqlServerDB.Database.SKU.Tier, *request.SqlServerDB.Database.SKU.Family)
	skuName := fmt.Sprintf("%d vCore", request.SqlServerDB.Database.SKU.Capacity)
	if *request.SqlServerDB.Database.Properties.ZoneRedundant {
		skuName += " Zone Redundancy"
	}

	readReplicaCostResponse, err := db.FindAzureSqlServerDatabaseReadReplicaCostComponentPrice(request.RegionCode, skuName, productName, "hours")
	if err != nil {
		return 0, err
	}
	return readReplicaCostResponse.Price, nil
}
