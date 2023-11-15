package aws

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"strings"
)

func RDSDBInstanceCostByResource(db *db.Database, request api.GetRDSInstanceRequest) (float64, error) {
	dbType := dbTypeMap[*request.DBInstance.DBInstance.Engine]
	licenseModel := licenseModelMap[*request.DBInstance.DBInstance.LicenseModel]

	deploymentOption := "Single-AZ"
	if request.DBInstance.DBInstance.MultiAZ {
		deploymentOption = "Multi-AZ"
	}

	dbInstanceCost, err := db.FindRDSInstancePrice(request.RegionCode, "dbinstance", dbType.engine,
		dbType.edition, licenseModel, deploymentOption, "Hrs")
	if err != nil {
		return 0, err
	}
	cost := dbInstanceCost.Price * costestimator.TimeInterval

	var volumeType string
	switch *request.DBInstance.DBInstance.StorageType {
	case "standard":
		volumeType = "Magnetic"
	case "io1", "io2":
		volumeType = "Provisioned IOPS"
	default:
		volumeType = "General Purpose"
	}

	storageCost, err := db.FindRDSDBStoragePrice(request.RegionCode, deploymentOption, volumeType, "GB")
	if err != nil {
		return 0, err
	}
	cost += storageCost.Price * costestimator.TimeInterval

	if strings.HasPrefix(*request.DBInstance.DBInstance.StorageType, "io") {
		IOPSCost, err := db.FindRDSDBIopsPrice(request.RegionCode, deploymentOption, "IOPS")
		if err != nil {
			return 0, err
		}
		cost += IOPSCost.Price * costestimator.TimeInterval
	}

	return cost, nil
}

type dbType struct {
	engine, edition string
}

var dbTypeMap = map[string]dbType{
	"aurora":            {"Aurora MySQL", ""},
	"aurora-postgresql": {"Aurora MySQL", ""},
	"mariadb":           {"MariaDB", ""},
	"postgresql":        {"MySQL", ""},
	"postgres":          {"PostgreSQL", ""},
	"oracle-se":         {"Oracle", "Standard"},
	"oracle-se1":        {"Oracle", "Standard One"},
	"oracle-se2":        {"Oracle", "Standard Two"},
	"oracle-ee":         {"Oracle", "Enterprise"},
	"sqlserver-se":      {"SQL Server", "Standard"},
	"sqlserver-ee":      {"SQL Server", "Enterprise"},
	"sqlserver-ex":      {"SQL Server", "Express"},
	"sqlserver-web":     {"SQL Server", "Web"},
}
var licenseModelMap = map[string]string{
	"license-included":       "License included",
	"bring-your-own-license": "Bring your own license",
}
