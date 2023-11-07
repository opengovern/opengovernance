package aws

import (
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
)

func RDSDBInstanceCostByResource(db *db.CostEstimatorDatabase, dbInstance es.RDSDBInstanceResponse) (float64, error) {
	description := dbInstance.Hits.Hits[0].Source.Description
	dbType := dbTypeMap[*description.DBInstance.Engine]
	licenseModel := licenseModelMap[*description.DBInstance.LicenseModel]

	deploymentOption := "Single-AZ"
	if description.DBInstance.MultiAZ {
		deploymentOption = "Multi-AZ"
	}

	dbInstanceCost, err := db.FindRDSInstancePrice(dbInstance.Hits.Hits[0].Source.Region, "dbinstance", dbType.engine,
		dbType.edition, licenseModel, deploymentOption, *description.DBInstance.StorageType)
	if err != nil {
		return 0, err
	}
	cost := dbInstanceCost.Price * TimeInterval
	return cost, nil
}

type dbType struct {
	engine, edition string
}

var dbTypeMap = map[string]dbType{
	"aurora":        {"Aurora MySQL", ""},
	"aurora-mysql":  {"Aurora MySQL", ""},
	"mariadb":       {"MariaDB", ""},
	"mysql":         {"MySQL", ""},
	"postgres":      {"PostgreSQL", ""},
	"oracle-se":     {"Oracle", "Standard"},
	"oracle-se1":    {"Oracle", "Standard One"},
	"oracle-se2":    {"Oracle", "Standard Two"},
	"oracle-ee":     {"Oracle", "Enterprise"},
	"sqlserver-se":  {"SQL Server", "Standard"},
	"sqlserver-ee":  {"SQL Server", "Enterprise"},
	"sqlserver-ex":  {"SQL Server", "Express"},
	"sqlserver-web": {"SQL Server", "Web"},
}
var licenseModelMap = map[string]string{
	"license-included":       "License included",
	"bring-your-own-license": "Bring your own license",
}
