package azure

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/labstack/gommon/log"
	"go.uber.org/zap"
	"math"
	"strings"
)

const (
	sqlServerlessTier = "general purpose"
	sqlHyperscaleTier = "hyperscale"
)

var (
	mssqlServiceTier = map[string]string{
		"GeneralPurpose":   "General Purpose",
		"BusinessCritical": "Business Critical",
		"Hyperscale":       "Hyperscale",
	}
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

func SqlServerDatabaseCostByResource(db *db.Database, request api.GetAzureSqlServersDatabasesRequest, logger *zap.Logger) (float64, error) {

	if strings.ToLower(*request.SqlServerDB.Database.SKU.Name) == "elasticpool" {
		logger.Info("Cost calculation in elasticPool purchasing model")
		cost, err := elasticPoolCostComponents(db, request, logger)
		if err != nil {
			return 0, err
		}
		return cost * costestimator.TimeInterval, nil
	}

	if request.SqlServerDB.Database.SKU.Capacity != nil {
		logger.Info("Cost calculation in Virtual core purchasing model ")
		vCoreCost, err := vCoreCostComponents(db, request, logger, kind)
		if err != nil {
			return 0, err
		}
		return vCoreCost * costestimator.TimeInterval, nil
	}

	logger.Info("Cost calculation in DTU purchasing model ")
	cost, err := dtuCostComponents(db, request, logger)
	if err != nil {
		return 0, err
	}

	return cost * costestimator.TimeInterval, nil
}

// ## perfect
func elasticPoolCostComponents(db *db.Database, request api.GetAzureSqlServersDatabasesRequest, logger *zap.Logger) (float64, error) {
	var cost float64
	longTermRetentionCost, err := longTermRetentionCostComponent(db, request, logger)
	if err != nil {
		return 0, err
	}
	cost += longTermRetentionCost

	pitrBackupCost, err := pitrBackupCostComponent(db, request, logger)
	if err != nil {
		return 0, err
	}
	cost += pitrBackupCost

	return cost, nil
}

func vCoreCostComponents(db *db.Database, request api.GetAzureSqlServersDatabasesRequest, logger *zap.Logger) (float64, error) {
	costComponents, err := computeHoursCostComponents(db, request)
	if err != nil {
		return 0, err
	}
	sku := request.SqlServerDB.Database.SKU

	if strings.ToLower(*sku.Tier) == sqlHyperscaleTier {
		log.Info("Cost calculation in the hyperscale tier type")

		ServiceTier, _ := mssqlServiceTier[*request.SqlServerDB.Database.SKU.Tier]
		productName := fmt.Sprintf("SQL Database SingleDB/Elastic Pool %s - Compute %s", ServiceTier, *request.SqlServerDB.Database.SKU.Family)
		skuName := fmt.Sprintf("%d vCore", request.SqlServerDB.Database.SKU.Capacity)
		if *request.SqlServerDB.Database.Properties.ZoneRedundant {
			skuName += " Zone Redundancy"
		}

		readReplicaCostResponse, err := db.FindAzureSqlServerDatabaseVCoreComponentsPrice(request.RegionCode, skuName, productName, "hours")
		if err != nil {
			log.Error("Error receiving hyperscale tier cost", zap.String("resourceId", fmt.Sprintf("%v", request.ResourceId)))
			return 0, err
		}

		costComponents += readReplicaCostResponse.Price
	}
	// ######################################### we have problem in product name here ###############################################
	if strings.ToLower(*sku.Tier) != sqlServerlessTier && !strings.Contains(*request.SqlServerDB.Database.Kind, "serverless") && strings.ToLower(string(*request.SqlServerDB.Database.Properties.LicenseType)) == "licenseincluded" {
		log.Info("Cost calculation where the tier is not equal to general purpose - serverless and license type is LicenseIncluded ")
		//it is wrong it should check
		productName := fmt.Sprintf("%s - %s", *request.SqlServerDB.Database.SKU.Tier, "SQL License")
		response, err := db.FindAzureSqlServerDatabaseVCoreForServerLessTierComponentPrice(request.RegionCode, "SQL Database", "Databases", productName, "vCore-hours")
		if err != nil {
			log.Error("Error receiving the cost where the tier is not equal to general purpose - serverless and license type is LicenseIncluded", zap.String("resourceId", fmt.Sprintf(request.ResourceId)))
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
	} else {
		storageTier, _ = mssqlServiceTier[*request.SqlServerDB.Database.SKU.Tier]
	}

	skuName := storageTier
	if *request.SqlServerDB.Database.Properties.ZoneRedundant {
		skuName += " Zone Redundancy"
	}
	// ######################################### we have problem in product name and metername here ###############################################
	productNameRegex := fmt.Sprintf("SQL Database %s - Storage", storageTier)
	StorageCostComponent, err := db.FindAzureSqlServerDatabasePrice(request.RegionCode, skuName, productNameRegex, "Data Stored", "GB")
	if err != nil {
		log.Error("Error receiving the storage cost component cost ", zap.String("resourceId", request.ResourceId))
		return 0, err
	}
	costComponents += StorageCostComponent.Price

	if strings.ToLower(*request.SqlServerDB.Database.SKU.Tier) != sqlHyperscaleTier {
		log.Info("Cost calculating where the tier is not equal to hyperscale ")
		longTermRetentionCost, err := longTermRetentionCostComponent(db, request, logger)
		if err != nil {
			return 0, err
		}
		costComponents += longTermRetentionCost

		pitrBackupCost, err := pitrBackupCostComponent(db, request, logger)
		if err != nil {
			return 0, err
		}
		costComponents += pitrBackupCost
	}
	return costComponents, nil
}

func dtuCostComponents(db *db.Database, request api.GetAzureSqlServersDatabasesRequest, logger *zap.Logger) (float64, error) {
	var cost float64
	skuName := strings.ToLower(*request.SqlServerDB.Database.SKU.Name)
	if skuName == "basic" {
		skuName = "b"
	}

	// we have problem here :
	productName := fmt.Sprintf("SQL Database Single %s", mssqlTierMapping[])
	meterName := fmt.Sprintf(" %v DTUs", request.SqlServerDB.Database.Properties.CurrentServiceObjectiveName)
	response, err := db.FindAzureSqlServerDatabasePrice(request.RegionCode, skuName, productName, meterName, "hours")
	if err != nil {
		log.Error(fmt.Errorf("Error in receiving compute %v cost ", request.SqlServerDB.Database.SKU.Name), zap.String("resourceId", request.ResourceId))
		return 0, err
	}
	cost += response.Price

	// we need to check ExtraStorageGB to see if we have that field
	// actually it should implement right here
	var extraStorageGB float64

	maxsizeGB := float64(*request.SqlServerDB.Database.Properties.MaxSizeBytes) / math.Pow(10, 9)

	if strings.HasPrefix(skuName, "b") {
		extraStorageGB = maxsizeGB
	} else if strings.HasPrefix(skuName, "s") {
		includedStorageGB := 250.0
		extraStorageGB = maxsizeGB - includedStorageGB
	} else if strings.HasPrefix(skuName, "p") {
		// we should not check if the extra size is bigger than the max storage that azure can support and if it was we send a message or something like that
		includedStorageGB, ok := mssqlStandardDTUIncludedStorage[*request.SqlServerDB.Database.Properties.CurrentServiceObjectiveName]
		if ok {
			extraStorageGB = maxsizeGB - includedStorageGB
		}
	}

	if extraStorageGB > 0 {
		tier := *request.SqlServerDB.Database.SKU.Tier
		if request.SqlServerDB.Database.SKU.Tier == nil {
			var ok bool
			tier, ok = mssqlTierMapping[strings.ToLower(*request.SqlServerDB.Database.SKU.Name)[:1]]
			if !ok {
				// what should put for resource address
				return 0, fmt.Errorf(fmt.Sprintf("Unrecognized tier for SKU '%s' for resource %s", *request.SqlServerDB.Database.SKU.Name, *request.SqlServerDB.Database.Name))
			}
		}
		productName = fmt.Sprintf("SQL Database %s - Storage", tier)
		// check tier in as sku name
		ExtraDataStorageResponse, err := db.FindAzureSqlServerDatabasePrice(request.RegionCode, tier, productName, "Data Stored", "GB")
		if err != nil {
			log.Error("Error receiving the extra storage cost component", zap.String("resourceId", request.ResourceId))
			return 0, err
		}
		cost += ExtraDataStorageResponse.Price
	}

	longTermRetentionCost, err := longTermRetentionCostComponent(db, request, logger)
	if err != nil {
		return 0, err
	}
	cost += longTermRetentionCost

	pitrBackupCost, err := pitrBackupCostComponent(db, request, logger)
	if err != nil {
		return 0, err
	}
	cost += pitrBackupCost

	return cost, nil
}

// ## perfect
func longTermRetentionCostComponent(dbFunc *db.Database, request api.GetAzureSqlServersDatabasesRequest, logger *zap.Logger) (float64, error) {
	logger.Info("Calculating cost in long term retention cost component")
	// mssqlStorageRedundancyTypeMapping should assign the GeoZone type
	redundancyType, ok := mssqlStorageRedundancyTypeMapping[strings.ToLower(string(*request.SqlServerDB.Database.Properties.CurrentBackupStorageRedundancy))]
	if !ok {
		redundancyType = "RA-GRS"
	}

	skuName := fmt.Sprintf("Backup %s", redundancyType)
	productName := fmt.Sprintln("SQL Database - LTR Backup Storage")
	meterName := fmt.Sprintf("%s Data Stored", skuName)

	longTermRetentionCost, err := dbFunc.FindAzureSqlServerDatabasePrice(request.RegionCode, skuName, productName, meterName, "GB")
	if err != nil {
		log.Error("Error receiving  long term retention cost has been failed", zap.String("resourceId", fmt.Sprintf("%v", request.ResourceId)))
		return 0, err
	}
	return longTermRetentionCost.Price, nil
}

// ## perfect
func pitrBackupCostComponent(dbFunc *db.Database, request api.GetAzureSqlServersDatabasesRequest, logger *zap.Logger) (float64, error) {
	logger.Info("Cost calculating PitrBackup component ")

	// mssqlStorageRedundancyTypeMapping should assign the GeoZone type
	redundancyType, ok := mssqlStorageRedundancyTypeMapping[strings.ToLower(string(*request.SqlServerDB.Database.Properties.CurrentBackupStorageRedundancy))]
	if !ok {
		redundancyType = "RA-GRS"
	}

	productName := fmt.Sprintln("SQL Database Single/Elastic Pool PITR Backup Storage")
	skuName := fmt.Sprintf("Backup %s", redundancyType)
	meterName := fmt.Sprintf("%s Data Stored", redundancyType)

	longTermRetentionCost, err := dbFunc.FindAzureSqlServerDatabasePrice(request.RegionCode, skuName, productName, meterName, "GB")
	if err != nil {
		logger.Error("Error receiving the Pitr Backup cost component has been failed", zap.String("resourceId", fmt.Sprintf("%v", request.ResourceId)))
		return 0, err
	}
	return longTermRetentionCost.Price, nil
}

func computeHoursCostComponents(db *db.Database, request api.GetAzureSqlServersDatabasesRequest) (float64, error) {
	log.Info("Cost calculating computeHours component")
	var cost float64
	if strings.ToLower(*request.SqlServerDB.Database.SKU.Tier) == sqlServerlessTier {
		productName := fmt.Sprintf("%s - %s", *request.SqlServerDB.Database.SKU.Tier, *request.SqlServerDB.Database.SKU.Family)
		// check the meter name
		response, err := db.FindAzureSqlServerDatabasePrice(request.RegionCode, "1 vCore", productName, "1 vCore - Free", "vCore-hours")
		if err != nil {
			log.Info("Error receiving the serverlessComputeHours cost component", zap.String("resourceId", request.ResourceId))
			return 0, err
		}
		cost += response.Price

		if *request.SqlServerDB.Database.Properties.ZoneRedundant {
			// we don't have any '1 vCore Zone Redundancy' sku name
			responseZoneRedundant, err := db.FindAzureSqlServerDatabasePrice(request.RegionCode, "1 vCore Zone Redundancy", productName, "Data Stored", "vCore-hours")
			if err != nil {
				log.Error("Error receiving the zoneRedundant serverlessComputeHours cost component", zap.String("resourceId", fmt.Sprintf("%v", request.ResourceId)))
				return 0, err
			}
			cost += responseZoneRedundant.Price
		}

		return cost, nil
	}

	productName := fmt.Sprintf("%s - Compute %s", *request.SqlServerDB.Database.SKU.Tier, *request.SqlServerDB.Database.SKU.Family)
	responseCost, err := db.FindAzureSqlServerDatabaseVCoreComponentsPrice(request.RegionCode, fmt.Sprintf("%d vCore", *request.SqlServerDB.Database.SKU.Capacity), productName, "hours")
	if err != nil {
		log.Error("Error receiving provisionedCompute cost component", zap.String("resourceId", fmt.Sprintf("%v", request.ResourceId)))
		return 0, err
	}
	cost += responseCost.Price

	if *request.SqlServerDB.Database.Properties.ZoneRedundant {
		ReadReplicaResponseCost, err := db.FindAzureSqlServerDatabaseVCoreComponentsPrice(request.RegionCode, fmt.Sprintf("%d vCore Zone Redundancy", *request.SqlServerDB.Database.SKU.Capacity), productName, "hours")
		if err != nil {
			log.Error("Error receiving ZoneRedundantProvisionedCompute cost component", zap.String("resourceId", fmt.Sprintf("%v", request.ResourceId)))
			return 0, err
		}
		cost += ReadReplicaResponseCost.Price
	}

	return cost, nil
}
