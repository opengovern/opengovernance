package model

import (
	"gorm.io/gorm"
	"strconv"
	"strings"
)

const (
	RDSDBStorageTier1Gp3BaseThroughput = 125.0
	RDSDBStorageTier1Gp3BaseIops       = 3000
	RDSDBStorageTier2Gp3BaseThroughput = 500.0
	RDSDBStorageTier2Gp3BaseIops       = 12000
	RDSDBStorageTier1Gp3SizeThreshold  = 400
)

type RDSDBStorage struct {
	gorm.Model

	// Basic fields
	RegionCode       string  `gorm:"index;type:citext"`
	DatabaseEngine   string  `gorm:"index;type:citext"`
	DatabaseEdition  string  `gorm:"index;type:citext"`
	PricePerUnit     float64 `gorm:"index:price_idx,sort:asc"`
	MinVolumeSizeGb  int32   `gorm:"index"`
	MaxVolumeSizeGb  int32   `gorm:"index"`
	MaxThroughputMB  float64 `gorm:"index"`
	MaxIops          int32   `gorm:"index"`
	VolumeType       string  `gorm:"index"`
	DeploymentOption string  `gorm:"index"`

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
	MinVolumeSize    string
	MaxVolumeSize    string
	EngineCode       string
	LicenseModel     string
	Group            string
	GroupDescription string
	UsageType        string
	Operation        string
	DeploymentModel  string
	LimitlessPreview string
	ServiceName      string
	VolumeName       string
}

type RDSDBStorageVolumeType string

const (
	RDSDBStorageVolumeTypeGP2                  RDSDBStorageVolumeType = "General Purpose"
	RDSDBStorageVolumeTypeGP3                  RDSDBStorageVolumeType = "General Purpose-GP3"
	RDSDBStorageVolumeTypeIO1                  RDSDBStorageVolumeType = "Provisioned IOPS"
	RDSDBStorageVolumeTypeIO2                  RDSDBStorageVolumeType = "Provisioned IOPS-IO2"
	RDSDBStorageVolumeTypeMagnetic             RDSDBStorageVolumeType = "Magnetic"
	RDSDBStorageVolumeTypeGeneralPurposeAurora RDSDBStorageVolumeType = "General Purpose-Aurora"
	RDSDBStorageVolumeTypeIOOptimizedAurora    RDSDBStorageVolumeType = "IO Optimized-Aurora"
)

var RDSDBStorageVolumeTypeToEBSType = map[string]string{
	string(RDSDBStorageVolumeTypeGP2):      "gp2",
	string(RDSDBStorageVolumeTypeGP3):      "gp3",
	string(RDSDBStorageVolumeTypeIO1):      "io1",
	string(RDSDBStorageVolumeTypeIO2):      "io2",
	string(RDSDBStorageVolumeTypeMagnetic): "standard",
	// Aurora not included as we don't know the mapping yet
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
				p.MinVolumeSizeGb = val * 1024
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
				p.MaxVolumeSizeGb = val * 1024
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
		case "Group Description":
			p.GroupDescription = row[index]
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

	// Computed fields
	if p.ProductFamily == "Database Storage" {
		engine := strings.ToLower(p.DatabaseEngine)
		volType := p.VolumeType
		// Using https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_Storage.html to fill in the iops/throughput values
		switch {
		case volType == string(RDSDBStorageVolumeTypeGP2) && !strings.Contains(engine, "aurora"): // GP2 non-aurora
			switch {
			case strings.Contains(engine, "mariadb"), strings.Contains(engine, "mysql"),
				strings.Contains(engine, "postgres"), strings.Contains(engine, "any"):
				p.MaxThroughputMB = 1000
				p.MaxIops = 64000
			case strings.Contains(engine, "oracle"):
				p.MaxThroughputMB = 1000
				p.MaxIops = 64000
			case strings.Contains(engine, "sql server"):
				p.MaxThroughputMB = 250
				p.MaxIops = 16000
			}
		case volType == string(RDSDBStorageVolumeTypeGP3) && !strings.Contains(engine, "aurora"): // GP3 non-aurora
			switch {
			case strings.Contains(engine, "db2"), strings.Contains(engine, "mariadb"),
				strings.Contains(engine, "mysql"), strings.Contains(engine, "postgres"), strings.Contains(engine, "any"):
				p.MaxThroughputMB = 4000
				p.MaxIops = 64000
			case strings.Contains(engine, "oracle"):
				p.MaxThroughputMB = 4000
				p.MaxIops = 64000
			case strings.Contains(engine, "sql server"):
				p.MaxThroughputMB = 1000
				p.MaxIops = 16000
			}
		case volType == string(RDSDBStorageVolumeTypeIO1) && !strings.Contains(engine, "aurora"): // IO1 non-aurora
			switch {
			case strings.Contains(engine, "db2"), strings.Contains(engine, "mariadb"),
				strings.Contains(engine, "mysql"), strings.Contains(engine, "postgres"), strings.Contains(engine, "any"):
				p.MaxThroughputMB = 4000
				p.MaxIops = 256000
			case strings.Contains(engine, "oracle"):
				p.MaxThroughputMB = 4000
				p.MaxIops = 256000
			case strings.Contains(engine, "sql server"):
				p.MaxThroughputMB = 1000
				p.MaxIops = 64000
			}
		case volType == string(RDSDBStorageVolumeTypeIO2) && !strings.Contains(engine, "aurora"): // IO2 non-aurora
			switch {
			case strings.Contains(engine, "db2"), strings.Contains(engine, "mariadb"),
				strings.Contains(engine, "mysql"), strings.Contains(engine, "postgres"), strings.Contains(engine, "any"):
				p.MaxThroughputMB = 4000
				p.MaxIops = 256000
			case strings.Contains(engine, "oracle"):
				p.MaxThroughputMB = 4000
				p.MaxIops = 256000
			case strings.Contains(engine, "sql server"):
				p.MaxThroughputMB = 4000
				p.MaxIops = 64000
			}
		case volType == string(RDSDBStorageVolumeTypeMagnetic) && !strings.Contains(engine, "aurora"): // Magnetic non-aurora
			p.MaxIops = 1000
			// This is an estimate, as the docs don't specify and leaving as 0 would make it unsuggestable (which you can make a case for)
			p.MaxThroughputMB = 100
		// aurora cases are not in the docs, so these are populated based on the general purpose and io optimized values
		// it shouldn't be too far off or matter too much as aurora is a managed service, and you can't change the storage type except between general purpose and io optimized
		// and for those we will use the cost and only cost to determine the cheapest option since other things are managed
		case volType == string(RDSDBStorageVolumeTypeGeneralPurposeAurora) && strings.Contains(engine, "aurora"): // General Purpose Aurora
			p.MaxThroughputMB = 4000
			p.MaxIops = 64000
		case volType == string(RDSDBStorageVolumeTypeIOOptimizedAurora) && strings.Contains(engine, "aurora"): // IO Optimized Aurora
			p.MaxThroughputMB = 4000
			p.MaxIops = 256000
		}
	}
}

func (p *RDSDBStorage) DoIngest() bool {
	if p.TermType != "OnDemand" ||
		p.LocationType == "AWS Outposts" ||
		p.VolumeType == "General Purpose (SSD)" ||
		p.VolumeType == "Provisioned IOPS (SSD)" {
		return false
	}
	if (p.ProductFamily == "Database Storage" && p.VolumeType == "General Purpose-GP3" && p.MinVolumeSize == "") ||
		(p.ProductFamily == "Database Storage" && p.VolumeType == "Provisioned IOPS-IO2" && p.MinVolumeSize == "") {
		return false
	}

	return true
}
