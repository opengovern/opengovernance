package model

import (
	"gorm.io/gorm"
	"strconv"
)

type RDSProduct struct {
	gorm.Model

	DatabaseEngine string  `gorm:"index;type:citext"`
	PricePerUnit   float64 `gorm:"index:price_idx,sort:asc"`

	// Basic fields
	SKU                        string
	OfferTermCode              string
	RateCode                   string
	TermType                   string
	PriceDescription           string
	EffectiveDate              string
	StartingRange              string
	EndingRange                string
	Unit                       string
	PricePerUnitStr            string
	Currency                   string
	RelatedTo                  string
	ProductFamily              string
	serviceCode                string
	Location                   string
	LocationType               string
	InstanceFamily             string
	vCPU                       string
	Storage                    string
	StorageMedia               string
	VolumeType                 string
	MinVolumeSize              string
	MaxVolumeSize              string
	EngineCode                 string
	DatabaseEdition            string
	LicenseModel               string
	DeploymentOption           string
	Group                      string
	GroupDescription           string
	usageType                  string
	operation                  string
	ACU                        string
	DeploymentModel            string
	EngineMajorVersion         string
	EngineMediaType            string
	ExtendedSupportPricingYear string
	InstanceTypeFamily         string
	LimitlessPreview           string
	NormalizationSizeFactor    string
	RegionCode                 string
	serviceName                string
	VolumeName                 string
}

func (p *RDSProduct) PopulateFromMap(columns map[string]int, row []string) {
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
			p.PricePerUnit, _ = strconv.ParseFloat(row[index], 64)
			p.PricePerUnitStr = row[index]
		case "Currency":
			p.Currency = row[index]
		case "RelatedTo":
			p.RelatedTo = row[index]
		case "Product Family":
			p.ProductFamily = row[index]
		case "serviceCode":
			p.serviceCode = row[index]
		case "Location":
			p.Location = row[index]
		case "Location Type":
			p.LocationType = row[index]
		case "Instance Family":
			p.InstanceFamily = row[index]
		case "vCPU":
			p.vCPU = row[index]
		case "Storage":
			p.Storage = row[index]
		case "Storage Media":
			p.StorageMedia = row[index]
		case "Volume Type":
			p.VolumeType = row[index]
		case "Min Volume Size":
			p.MinVolumeSize = row[index]
		case "Max Volume Size":
			p.MaxVolumeSize = row[index]
		case "Engine Code":
			p.EngineCode = row[index]
		case "Database Engine":
			p.DatabaseEngine = row[index]
		case "Database Edition":
			p.DatabaseEdition = row[index]
		case "License Model":
			p.LicenseModel = row[index]
		case "Deployment Option":
			p.DeploymentOption = row[index]
		case "Group":
			p.Group = row[index]
		case "Group Description":
			p.GroupDescription = row[index]
		case "usageType":
			p.usageType = row[index]
		case "operation":
			p.operation = row[index]
		case "ACU":
			p.ACU = row[index]
		case "Deployment Model":
			p.DeploymentModel = row[index]
		case "Engine Major Version":
			p.EngineMajorVersion = row[index]
		case "Engine Media Type":
			p.EngineMediaType = row[index]
		case "Extended Support Pricing Year":
			p.ExtendedSupportPricingYear = row[index]
		case "Instance Type Family":
			p.InstanceTypeFamily = row[index]
		case "LimitlessPreview":
			p.LimitlessPreview = row[index]
		case "Normalization Size Factor":
			p.NormalizationSizeFactor = row[index]
		case "Region Code":
			p.RegionCode = row[index]
		case "serviceName":
			p.serviceName = row[index]
		case "Volume Name":
			p.VolumeName = row[index]
		}
	}
}
