package azure

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/product"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/query"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/util"
	"github.com/shopspring/decimal"
	"math"
	"strings"
)

const (
	sqlServerlessTier = "general purpose"
	sqlHyperscaleTier = "hyperscale"
)

var (
	HourToMonthUnitMultiplier = decimal.NewFromInt(730)
)

var (
	isElasticPool bool

	sqlFamilyMapping = map[string]string{
		"gen5": "Compute Gen5",
		"gen4": "Compute Gen4",
		"m":    "Compute M Series",
	}
	sqlTierMapping = map[string]string{
		"GeneralPurpose":           "General Purpose",
		"GeneralPurposeServerless": "General Purpose - Serverless",
		"Hyperscale":               "Hyperscale",
		"BusinessCritical":         "Business Critical",
	}

	dtuMap = dtuMapping{
		"free":  true,
		"basic": true,

		"s": true, // covers Standard, System editions
		"d": true, // covers DataWarehouse editions
		"p": true, // covers Premium editions
	}
)

type dtuMapping map[string]bool

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

// SqlServerDB is the entity that holds the logic to calculate price
// of the azure_sql_serverdatabase
type SqlServerDB struct {
	provider *Provider

	location                       string
	skuName                        string
	skuCapacity                    int32
	skuFamily                      string
	tier                           string
	zoneRedundant                  bool
	kind                           string
	licenseType                    string
	maxSizeBytes                   int64
	readScale                      string
	elasticPoolId                  *string
	monthlyVCoreHours              int64
	extraDataStorageGB             float64
	longTermRetentionStorageGB     int64
	backupStorageGB                int64
	currentServiceObjectiveName    string
	currentBackupStorageRedundancy string
}

// SqlServerDBValues is holds the values that we need to be able
// to calculate the price of the Sql ServerDB Values
type SqlServerDBValues struct {
	Location                       string  `json:"location"`
	SkuName                        string  `json:"sku_name"`
	SkuCapacity                    int32   `json:"sku_capacity"`
	SkuFamily                      string  `json:"sku_family"`
	Tier                           string  `json:"tier"`
	ZoneRedundant                  bool    `json:"zone_redundant"`
	Kind                           string  `json:"kind"`
	LicenseType                    string  `json:"license_type"`
	MaxSizeBytes                   int64   `json:"max_size_bytes"`
	ReadScale                      string  `json:"read_scale"`
	ElasticPoolId                  *string `json:"elastic_pool_id"`
	MonthlyVCoreHours              int64   `json:"monthly_vcore_hours"`
	ExtraDataStorageGB             float64 `json:"extra_data_storage_gb"`
	LongTermRetentionStorageGB     int64   `json:"long_term_retention_storage_gb"`
	BackupStorageGB                int64   `json:"backup_storage_gb"`
	CurrentServiceObjectiveName    string  `json:"current_service_objective_name"`
	CurrentBackupStorageRedundancy string  `json:"current_backup_storage_redundancy"`
}

// decodeSqlServerDB decodes and returns sql serverDB values from a Terraform values map.
func decodeSqlServerDB(request api.GetAzureSqlServersDatabasesRequest, monthlyVCoreHours int64, extraDataStorageGB float64, longTermRetentionStorageGB int64, backupStorageGB int64) SqlServerDBValues {
	return SqlServerDBValues{
		Location:                       *request.SqlServerDB.Database.Location,
		SkuName:                        *request.SqlServerDB.Database.SKU.Name,
		SkuFamily:                      *request.SqlServerDB.Database.SKU.Family,
		SkuCapacity:                    *request.SqlServerDB.Database.SKU.Capacity,
		Tier:                           *request.SqlServerDB.Database.SKU.Tier,
		ZoneRedundant:                  *request.SqlServerDB.Database.Properties.ZoneRedundant,
		Kind:                           *request.SqlServerDB.Database.Kind,
		LicenseType:                    string(*request.SqlServerDB.Database.Properties.LicenseType),
		MaxSizeBytes:                   *request.SqlServerDB.Database.Properties.MaxSizeBytes,
		ReadScale:                      string(*request.SqlServerDB.Database.Properties.ReadScale),
		ElasticPoolId:                  request.SqlServerDB.Database.Properties.ElasticPoolID,
		MonthlyVCoreHours:              monthlyVCoreHours,
		ExtraDataStorageGB:             extraDataStorageGB,
		LongTermRetentionStorageGB:     longTermRetentionStorageGB,
		BackupStorageGB:                backupStorageGB,
		CurrentServiceObjectiveName:    *request.SqlServerDB.Database.Properties.CurrentServiceObjectiveName,
		CurrentBackupStorageRedundancy: string(*request.SqlServerDB.Database.Properties.CurrentBackupStorageRedundancy),
	}
}

// newSqlServerDB initializes a new SqlServerDB from the provider
func (p *Provider) newSqlServerDB(vals SqlServerDBValues) *SqlServerDB {
	inst := &SqlServerDB{
		provider: p,

		location:                       vals.Location,
		skuName:                        vals.SkuName,
		skuFamily:                      vals.SkuFamily,
		skuCapacity:                    vals.SkuCapacity,
		tier:                           vals.Tier,
		zoneRedundant:                  vals.ZoneRedundant,
		kind:                           vals.Kind,
		licenseType:                    vals.LicenseType,
		maxSizeBytes:                   vals.MaxSizeBytes,
		readScale:                      vals.ReadScale,
		elasticPoolId:                  vals.ElasticPoolId,
		monthlyVCoreHours:              vals.MonthlyVCoreHours,
		extraDataStorageGB:             vals.ExtraDataStorageGB,
		longTermRetentionStorageGB:     vals.LongTermRetentionStorageGB,
		backupStorageGB:                vals.BackupStorageGB,
		currentServiceObjectiveName:    vals.CurrentServiceObjectiveName,
		currentBackupStorageRedundancy: vals.CurrentBackupStorageRedundancy,
	}

	return inst
}

func (d dtuMapping) usesDTUUnits(skuName string) bool {
	sanitized := strings.ToLower(skuName)
	if d[sanitized] {
		return true
	}

	if sanitized == "" {
		return false
	}

	return d[sanitized[0:1]]
}

// SQLDatabase splits pricing into two different models. DTU & vCores.
//
//	Database Transaction Unit (DTU) is made a performance metric representing a mixture of performance metrics
//	in Azure SQL. Some include: CPU, I/O, Memory. DTU is used as Azure tries to simplify billing by using a single metric.
//
//	Virtual Core (vCore) pricing is designed to translate from on premise hardware metrics (cores) into the cloud
//	SQL instance. vCore is designed to allow users to better estimate their resource limits, e.g. RAM.
//
// SQL databases that follow a DTU pricing model have the following costs associated with them:
//
//  1. Costs based on the number of DTUs that the sql database has
//  2. Extra backup data costs - this is configured using SQLDatabase.ExtraDataStorageGB
//  3. Long term data backup costs - this is configured using SQLDatabase.LongTermRetentionStorageGB
//
// SQL databases that follow a vCore pricing model have the following costs associated with them:
//
//  1. Costs based on the number of vCores the resource has
//  2. Extra pricing if any database read replicas have been provisioned
//  3. Additional charge for SQL Server licensing based on vCores amount
//  4. Charges for storage used
//  5. Charges for long term data backup - this is configured using SQLDatabase.LongTermRetentionStorageGB

// Components returns the price component queries that make up this Instance.
func (inst *SqlServerDB) Components() []query.Component {
	if strings.ToLower(inst.skuName) == "elasticpool" || inst.elasticPoolId != nil {
		isElasticPool = true
	} else if !dtuMap.usesDTUUnits(inst.skuName) {
		familyKey := strings.ToLower(inst.skuFamily)
		family, ok := sqlFamilyMapping[familyKey]
		if !ok {
			// TODO : i should set a error in here like : "Invalid family in MSSQL SKU for resource %s: %s", address, sku
			return nil
		}
		inst.skuFamily = family
	}

	tier, ok := mssqlServiceTier[inst.tier]
	if ok {
		inst.tier = tier
	}

	maxSizeGB := float64(inst.maxSizeBytes) / math.Pow(10, 9)

	if isElasticPool {
		return elasticPoolCostComponents(inst.currentBackupStorageRedundancy)
	}

	if inst.skuCapacity != 0 {
		return vCoreCostComponents(inst, maxSizeGB)
	}

	return dtuCostComponents(inst, maxSizeGB)
}

// elasticPoolCostComponents calculate elasticPool deployment type costs
func elasticPoolCostComponents(currentBackupStorageRedundancy string) []query.Component {
	var costComponents []query.Component

	costComponents = append(costComponents, longTermRetentionCostComponent(0, currentBackupStorageRedundancy))
	costComponents = append(costComponents, pitrBackupCostComponent(0, currentBackupStorageRedundancy))

	return costComponents
}

// vCoreCostComponents calculate vCore purchase model costs
func vCoreCostComponents(inst *SqlServerDB, maxsizeGB float64) []query.Component {
	var costComponents []query.Component
	var replicaCount decimal.Decimal
	if inst.readScale == "Enabled" {
		replicaCount = decimal.NewFromInt(1)
	}

	computeHoursComponent := computeHoursCostComponents(inst.tier, inst.skuFamily, inst.skuCapacity, inst.skuName, inst.kind,
		inst.zoneRedundant, inst.monthlyVCoreHours)
	for i := 0; i < len(computeHoursComponent); i++ {
		costComponents = append(costComponents, computeHoursComponent[i])
	}

	if strings.ToLower(inst.tier) == sqlHyperscaleTier {

		ServiceTier, _ := mssqlServiceTier[inst.tier]
		productName := fmt.Sprintf("/%s - %s/", ServiceTier, inst.skuFamily)

		HsSkuName := fmt.Sprintf("%d vCore", inst.skuCapacity)
		if inst.zoneRedundant {
			HsSkuName += " Zone Redundancy"
		}

		costComponents = append(costComponents, query.Component{
			Name:           "Read replicas",
			Unit:           "hours",
			HourlyQuantity: replicaCount,
			ProductFilter: &product.Filter{
				AttributeFilters: []*product.AttributeFilter{
					{Key: "productName", ValueRegex: util.StringPtr(productName)},
					{Key: "skuName", Value: util.StringPtr(HsSkuName)},
				},
			},
		})

	}

	if strings.ToLower(inst.tier) != sqlServerlessTier && !strings.Contains(inst.kind, "serverless") && strings.ToLower(inst.licenseType) == "licenseincluded" {
		costComponents = append(costComponents, sqlLicenseCostComponent(inst.location, inst.tier, inst.skuCapacity))
	}

	costComponents = append(costComponents, mssqlStorageCostComponent(inst.tier, inst.zoneRedundant, maxsizeGB))

	if strings.ToLower(inst.tier) != sqlHyperscaleTier {
		costComponents = append(costComponents, longTermRetentionCostComponent(inst.longTermRetentionStorageGB, inst.currentBackupStorageRedundancy))
		costComponents = append(costComponents, pitrBackupCostComponent(inst.backupStorageGB, inst.currentBackupStorageRedundancy))
	}

	return costComponents
}

// dtuCostComponents calculate DTU purchase model costs
func dtuCostComponents(inst *SqlServerDB, maxsizeGB float64) []query.Component {
	var costComponents []query.Component

	skuName := strings.ToLower(inst.skuName)
	if skuName == "basic" {
		skuName = "b"
	}
	daysInMonth := HourToMonthUnitMultiplier.DivRound(decimal.NewFromInt(24), 24)

	costComponents = append(costComponents, query.Component{
		Name:            fmt.Sprintf("Compute (%s)", strings.ToTitle(inst.skuName)),
		Unit:            "hour",
		MonthlyQuantity: daysInMonth,
		ProductFilter: &product.Filter{
			AttributeFilters: []*product.AttributeFilter{
				{Key: "productName", ValueRegex: util.StringPtr("^SQL Database Single")},
				{Key: "skuName", ValueRegex: util.StringPtr(fmt.Sprintf("^%s$", skuName))},
				{Key: "meterName", ValueRegex: util.StringPtr("DTU(s)?$")},
			},
		},
	})

	var extraStorageGB float64
	if strings.HasPrefix(skuName, "b") && inst.extraDataStorageGB != 0 {
		extraStorageGB = inst.extraDataStorageGB
	} else if strings.HasPrefix(skuName, "s") && maxsizeGB != 0 {
		includedStorageGB := 250.0
		extraStorageGB = maxsizeGB - includedStorageGB
	} else if strings.HasPrefix(skuName, "p") && maxsizeGB != 0 {
		includedStorageGB, ok := mssqlStandardDTUIncludedStorage[inst.currentServiceObjectiveName]
		if ok {
			extraStorageGB = maxsizeGB - includedStorageGB
		}
	}

	if extraStorageGB > 0 {

		tier := inst.tier
		if tier == "" {
			var ok bool
			tier, ok = mssqlTierMapping[strings.ToLower(inst.skuName)[:1]]
			if !ok {
				// TODO : we should put a error or log in here
				//				log.Warn().Msgf("Unrecognized tier for SKU '%s' for resource %s", r.SKU, r.Address)
				return nil
			}
		}

		c := &query.Component{
			Name:            "Extra data storage",
			Unit:            "GB",
			MonthlyQuantity: decimal.NewFromFloat(extraStorageGB),
			ProductFilter: &product.Filter{
				AttributeFilters: []*product.AttributeFilter{
					{Key: "productName", ValueRegex: util.StringPtr(fmt.Sprintf("/SQL Database %s - Storage/i", tier))},
					{Key: "skuName", ValueRegex: util.StringPtr(fmt.Sprintf("/^%s$/i", tier))},
					{Key: "meterName", Value: util.StringPtr("Data Stored")},
				},
			},
		}
		if c != nil {
			costComponents = append(costComponents, *c)
		}
	}

	costComponents = append(costComponents, longTermRetentionCostComponent(inst.longTermRetentionStorageGB, inst.currentBackupStorageRedundancy))
	costComponents = append(costComponents, pitrBackupCostComponent(inst.backupStorageGB, inst.currentBackupStorageRedundancy))

	return costComponents
}

// longTermRetentionCostComponent is a component for the elasticPool and vCore and dtu sql serverDB components
func longTermRetentionCostComponent(longTermRetentionStorageGB int64, currentBackupStorageRedundancy string) query.Component {
	var retention decimal.Decimal
	if longTermRetentionStorageGB != 0 {
		retention = decimal.NewFromInt(longTermRetentionStorageGB)
	}

	// TODO: mssqlStorageRedundancyTypeMapping should assign the GeoZone type
	redundancyType, ok := mssqlStorageRedundancyTypeMapping[strings.ToLower(currentBackupStorageRedundancy)]
	if !ok {
		redundancyType = "RA-GRS"
	}

	skuName := fmt.Sprintf("Backup %s", redundancyType)
	meterName := fmt.Sprintf("%s Data Stored", skuName)

	return query.Component{
		Name:            fmt.Sprintf("Long-term retention (%s)", redundancyType),
		MonthlyQuantity: retention,
		Unit:            "GB",
		ProductFilter: &product.Filter{
			AttributeFilters: []*product.AttributeFilter{
				{Key: "skuName", Value: util.StringPtr(skuName)},
				{Key: "productName", Value: util.StringPtr(fmt.Sprintln("SQL Database - LTR Backup Storage"))},
				{Key: "meterName", ValueRegex: util.StringPtr(meterName)},
			},
		},
	}
}

// pitrBackupCostComponent is a component for the elasticPool and vCore and dtu sql serverDB components
func pitrBackupCostComponent(backupStorageGB int64, currentBackupStorageRedundancy string) query.Component {
	var pitrGB decimal.Decimal
	if backupStorageGB != 0 {
		pitrGB = decimal.NewFromInt(backupStorageGB)
	}

	// TODO: mssqlStorageRedundancyTypeMapping should assign the GeoZone type
	redundancyType, ok := mssqlStorageRedundancyTypeMapping[strings.ToLower(currentBackupStorageRedundancy)]
	if !ok {
		redundancyType = "RA-GRS"
	}

	return query.Component{
		Name:            fmt.Sprintf("PITR backup storage (%s)", redundancyType),
		Unit:            "GB",
		MonthlyQuantity: pitrGB,
		ProductFilter: &product.Filter{
			AttributeFilters: []*product.AttributeFilter{
				{Key: "productName", ValueRegex: util.StringPtr("PITR Backup Storage")},
				{Key: "skuName", Value: util.StringPtr(fmt.Sprintf("Backup %s", redundancyType))},
				{Key: "meterName", ValueRegex: util.StringPtr(fmt.Sprintf("%s Data Stored", redundancyType))},
			},
		},
	}
}

// computeHoursCostComponents is a component for the vCore sql serverDB component
func computeHoursCostComponents(tier string, skuFamily string, skuCapabilities int32, skuName string, kind string, zoneRedundant bool, monthlyVCoreHours int64) []query.Component {
	if strings.ToLower(tier) == sqlServerlessTier && !strings.Contains(kind, "serverless") {
		return serverlessComputeHoursCostComponents(tier, skuFamily, skuName, zoneRedundant, monthlyVCoreHours)
	}
	return provisionedComputeCostComponents(tier, skuName, skuFamily, skuCapabilities, zoneRedundant)
}

// serverlessComputeHoursCostComponents is a component for the computeHours sql serverDB component
func serverlessComputeHoursCostComponents(tier string, skuFamily string, skuName string, zoneRedundant bool, monthlyVCoreHours int64) []query.Component {
	var costComponents []query.Component
	var vCoreHours decimal.Decimal

	if monthlyVCoreHours != 0 {
		vCoreHours = decimal.NewFromInt(monthlyVCoreHours)
	}

	costComponents = append(costComponents, query.Component{
		Name:            fmt.Sprintf("Compute (serverless, %s)", skuName),
		Unit:            "vCore-hours",
		MonthlyQuantity: vCoreHours,
		ProductFilter: &product.Filter{
			AttributeFilters: []*product.AttributeFilter{
				{Key: "productName", ValueRegex: util.StringPtr(fmt.Sprintf("%s - %s", tier, skuFamily))},
				{Key: "skuName", Value: util.StringPtr("1 vCore")},
				{Key: "meterName", ValueRegex: util.StringPtr("^(?!.* - Free$).*$")},
			},
		},
	})

	if zoneRedundant {

		costComponents = append(costComponents, query.Component{
			Name:            fmt.Sprintf("Zone redundancy (serverless, %s)", skuName),
			Unit:            "vCore-hours",
			MonthlyQuantity: vCoreHours,
			ProductFilter: &product.Filter{
				AttributeFilters: []*product.AttributeFilter{
					{Key: "productName", ValueRegex: util.StringPtr(fmt.Sprintf("%s - %s", tier, skuFamily))},
					{Key: "skuName", Value: util.StringPtr("1 vCore Zone Redundancy")},
					{Key: "meterName", ValueRegex: util.StringPtr("^(?!.* - Free$).*$")},
				},
			},
		})

	}

	return costComponents
}

// provisionedComputeCostComponents is a component for the computeHours sql serverDB component
func provisionedComputeCostComponents(tier string, skuName string, skuFamily string, skuCapabilities int32, zoneRedundant bool) []query.Component {
	var costComponents []query.Component

	productName := fmt.Sprintf("%s - %s", tier, skuFamily)

	costComponents = append(costComponents, query.Component{
		Name:           fmt.Sprintf("Compute (provisioned, %s)", skuName),
		Unit:           "hours",
		HourlyQuantity: decimal.NewFromInt(1),
		ProductFilter: &product.Filter{
			AttributeFilters: []*product.AttributeFilter{
				{Key: "productName", ValueRegex: util.StringPtr(productName)},
				{Key: "skuName", Value: util.StringPtr(fmt.Sprintf("%d vCore", skuCapabilities))},
			},
		},
	})

	if zoneRedundant {
		costComponents = append(costComponents, query.Component{
			Name:           fmt.Sprintf("Zone redundancy (provisioned, %s)", skuName),
			Unit:           "hours",
			HourlyQuantity: decimal.NewFromInt(1),
			ProductFilter: &product.Filter{
				AttributeFilters: []*product.AttributeFilter{
					{Key: "productName", ValueRegex: util.StringPtr(productName)},
					{Key: "skuName", Value: util.StringPtr(fmt.Sprintf("%d vCore Zone Redundancy", skuCapabilities))},
				},
			},
		})
	}

	return costComponents
}

// sqlLicenseCostComponent is a component for the vCore sql serverDB component
func sqlLicenseCostComponent(region string, tier string, skuCapacity int32) query.Component {
	productName := fmt.Sprintf("%s - %s", tier, "SQL License")

	licenseRegion := "Global"
	if strings.Contains(region, "usgov") {
		licenseRegion = "US Gov"
	}

	if strings.Contains(region, "china") {
		licenseRegion = "China"
	}

	if strings.Contains(region, "germany") {
		licenseRegion = "Germany"
	}

	coresVal := int32(1)
	if skuCapacity != 0 {
		coresVal = skuCapacity
	}
	vendorName := "azure"

	return query.Component{
		Name:           "SQL license",
		Unit:           "vCore-hours",
		HourlyQuantity: decimal.NewFromInt(int64(coresVal)),
		ProductFilter: &product.Filter{
			Provider: &vendorName,
			Location: util.StringPtr(licenseRegion),
			Family:   util.StringPtr("Databases"),
			Service:  util.StringPtr("SQL Database"),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "productName", ValueRegex: util.StringPtr(productName)},
			},
		},
	}
}

// mssqlStorageCostComponent is a component for the vCore sql serverDB component
func mssqlStorageCostComponent(tier string, zoneRedundant bool, maxsizeGB float64) query.Component {
	storageGB := decimal.NewFromInt(5)
	if maxsizeGB != 0 {
		storageGB = decimal.NewFromFloat(maxsizeGB)
	}

	storageTier := tier
	if strings.ToLower(storageTier) == "general purpose" {
		storageTier = "General Purpose"
	} else {
		storageTier, _ = mssqlServiceTier[tier]
	}

	skuName := tier
	if zoneRedundant {
		skuName += " Zone Redundancy"
	}

	productNameRegex := fmt.Sprintf("/%s - Storage/", storageTier)
	return query.Component{
		Name:            "Storage",
		Unit:            "GB",
		MonthlyQuantity: storageGB,
		ProductFilter: &product.Filter{
			AttributeFilters: []*product.AttributeFilter{
				{Key: "productName", ValueRegex: util.StringPtr(productNameRegex)},
				{Key: "skuName", Value: util.StringPtr(skuName)},
				{Key: "meterName", ValueRegex: util.StringPtr("Data Stored$")},
			},
		},
	}
}
