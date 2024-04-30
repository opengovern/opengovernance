package model

import (
	"gorm.io/gorm"
)

type RDSDBStorage struct {
	gorm.Model

	// Basic fields
	SKU              string
	OfferTermCode    string
	RateCode         string
	TermType         string
	PriceDescription string
	EffectiveDate    string
	StartingRange    string
	EndingRange      string
	Unit             string
	PricePerUnit     string
	Currency         string
	ProductFamily    string
	serviceCode      string
	Location         string
	LocationType     string
	StorageMedia     string
	VolumeType       string
	MinVolumeSize    string
	MaxVolumeSize    string
	EngineCode       string
	DatabaseEngine   string
	DatabaseEdition  string
	LicenseModel     string
	DeploymentOption string
	Group            string
	usageType        string
	operation        string
	DeploymentModel  string
	LimitlessPreview string
	RegionCode       string
	serviceName      string
	VolumeName       string
}

func (p RDSDBStorage) PopulateFromMap(columns map[string]int, row []string) {
	for col, index := range columns {
		switch col {
		case "SKU":
			p.SKU = row[index]
		case "OfferTermCode":
			p.OfferTermCode = row[index]
		case "RateCode":
			p.RateCode = row[index]
		case "TermType":
			p.TermType = row[index]
		case "PriceDescription":
			p.PriceDescription = row[index]
		case "EffectiveDate":
			p.EffectiveDate = row[index]
		case "StartingRange":
			p.StartingRange = row[index]
		case "EndingRange":
			p.EndingRange = row[index]
		case "Unit":
			p.Unit = row[index]
		case "PricePerUnit":
			p.PricePerUnit = row[index]
		case "Currency":
			p.Currency = row[index]
		case "ProductFamily":
			p.ProductFamily = row[index]
		case "serviceCode":
			p.serviceCode = row[index]
		case "Location":
			p.Location = row[index]
		case "LocationType":
			p.LocationType = row[index]
		case "StorageMedia":
			p.StorageMedia = row[index]
		case "VolumeType":
			p.VolumeType = row[index]
		case "MinVolumeSize":
			p.MinVolumeSize = row[index]
		case "MaxVolumeSize":
			p.MaxVolumeSize = row[index]
		case "EngineCode":
			p.EngineCode = row[index]
		case "DatabaseEngine":
			p.DatabaseEngine = row[index]
		case "DatabaseEdition":
			p.DatabaseEdition = row[index]
		case "LicenseModel":
			p.LicenseModel = row[index]
		case "DeploymentOption":
			p.DeploymentOption = row[index]
		case "Group":
			p.Group = row[index]
		case "usageType":
			p.usageType = row[index]
		case "operation":
			p.operation = row[index]
		case "DeploymentModel":
			p.DeploymentModel = row[index]
		case "LimitlessPreview":
			p.LimitlessPreview = row[index]
		case "RegionCode":
			p.RegionCode = row[index]
		case "serviceName":
			p.serviceName = row[index]
		case "VolumeName":
			p.VolumeName = row[index]
		}
	}
}
