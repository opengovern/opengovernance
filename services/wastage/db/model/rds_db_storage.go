package model

import (
	"gorm.io/gorm"
	"strconv"
	"strings"
)

type RDSDBStorage struct {
	gorm.Model

	// Basic fields
	RegionCode      string  `gorm:"index;type:citext"`
	DatabaseEngine  string  `gorm:"index;type:citext"`
	DatabaseEdition string  `gorm:"index;type:citext"`
	PricePerUnit    float64 `gorm:"index:price_idx,sort:asc"`
	MinVolumeSizeGb int32   `gorm:"index"`
	MaxVolumeSizeGb int32   `gorm:"index"`

	SKU              string
	OfferTermCode    string
	RateCode         string
	TermType         string
	PriceDescription string
	EffectiveDate    string
	StartingRange    string
	EndingRange      string
	Unit             string
	PricePerUnitStr  string
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
	LicenseModel     string
	DeploymentOption string
	Group            string
	UsageType        string
	Operation        string
	DeploymentModel  string
	LimitlessPreview string
	ServiceName      string
	VolumeName       string
}

func (p *RDSDBStorage) PopulateFromMap(columns map[string]int, row []string) {
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
		case "Product Family":
			p.ProductFamily = row[index]
		case "serviceCode":
			p.serviceCode = row[index]
		case "Location":
			p.Location = row[index]
		case "Location Type":
			p.LocationType = row[index]
		case "Storage Media":
			p.StorageMedia = row[index]
		case "Volume Type":
			p.VolumeType = row[index]
		case "Min Volume Size":
			var val int32
			var unit string
			for _, c := range strings.Split(strings.ToLower(row[index]), " ") {
				v, err := strconv.ParseInt(c, 10, 32)
				if err == nil {
					val = max(val, int32(v))
					break
				}
				if strings.Contains(c, "gb") || strings.Contains(c, "tb") {
					unit = c
					break
				}
			}
			switch unit {
			case "gb":
				p.MinVolumeSizeGb = val
			case "tb":
				p.MinVolumeSizeGb = val * 1000
			default:
				p.MinVolumeSizeGb = val
			}
			p.MinVolumeSize = row[index]
		case "Max Volume Size":
			var val int32
			var unit string
			for _, c := range strings.Split(strings.ToLower(row[index]), " ") {
				v, err := strconv.ParseInt(c, 10, 32)
				if err == nil {
					val = max(val, int32(v))
					break
				}
				if strings.Contains(c, "gb") || strings.Contains(c, "tb") {
					unit = c
					break
				}
			}
			switch unit {
			case "gb":
				p.MaxVolumeSizeGb = val
			case "tb":
				p.MaxVolumeSizeGb = val * 1000
			default:
				p.MaxVolumeSizeGb = val
			}
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
		case "usageType":
			p.UsageType = row[index]
		case "operation":
			p.Operation = row[index]
		case "Deployment Model":
			p.DeploymentModel = row[index]
		case "LimitlessPreview":
			p.LimitlessPreview = row[index]
		case "Region Code":
			p.RegionCode = row[index]
		case "serviceName":
			p.ServiceName = row[index]
		case "Volume Name":
			p.VolumeName = row[index]
		}
	}
}
