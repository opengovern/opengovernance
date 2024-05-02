package model

import (
	"gorm.io/gorm"
	"strconv"
	"strings"
)

type RDSDBInstance struct {
	gorm.Model

	// Basic fields

	VCpu              int64    `gorm:"index"`
	MemoryGb          int64    `gorm:"index"`
	NetworkThroughput *float64 `gorm:"index"` // In bytes/s
	DatabaseEngine    string   `gorm:"index"`
	DeploymentOption  string   `gorm:"index"`

	SKU                         string
	OfferTermCode               string
	RateCode                    string
	TermType                    string
	PriceDescription            string
	EffectiveDate               string
	StartingRange               string
	EndingRange                 string
	Unit                        string
	PricePerUnit                string
	Currency                    string
	ProductFamily               string
	serviceCode                 string
	Location                    string
	LocationType                string
	InstanceType                string
	CurrentGeneration           string
	InstanceFamily              string
	PhysicalProcessor           string
	ClockSpeed                  string
	Memory                      string
	Storage                     string
	NetworkPerformance          string
	ProcessorArchitecture       string
	EngineCode                  string
	DatabaseEdition             string
	LicenseModel                string
	usageType                   string
	operation                   string
	DedicatedEBSThroughput      string
	DeploymentModel             string
	EngineMediaType             string
	EnhancedNetworkingSupported string
	InstanceTypeFamily          string
	NormalizationSizeFactor     string
	PricingUnit                 string
	ProcessorFeatures           string
	RegionCode                  string
	serviceName                 string
}

func (p *RDSDBInstance) PopulateFromMap(columns map[string]int, row []string) {
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
		case "InstanceType":
			p.InstanceType = row[index]
		case "CurrentGeneration":
			p.CurrentGeneration = row[index]
		case "InstanceFamily":
			p.InstanceFamily = row[index]
		case "vCPU":
			i, err := strconv.ParseInt(row[index], 10, 64)
			if err == nil {
				p.VCpu = i
			}
		case "PhysicalProcessor":
			p.PhysicalProcessor = row[index]
		case "ClockSpeed":
			p.ClockSpeed = row[index]
		case "Memory":
			p.Memory = row[index]
			for _, part := range strings.Split(row[index], " ") {
				i, err := strconv.ParseInt(part, 10, 64)
				if err == nil {
					p.MemoryGb = max(p.MemoryGb, i)
				}
			}
		case "Storage":
			p.Storage = row[index]
		case "NetworkPerformance":
			p.NetworkPerformance = row[index]
			for _, part := range strings.Split(row[index], " ") {
				i, err := strconv.ParseFloat(part, 64)
				// convert from Gbps to bytes/s
				i = i * 1e9 / 8
				if err == nil {
					if p.NetworkThroughput == nil {
						p.NetworkThroughput = &i
					} else {
						*p.NetworkThroughput = max(*p.NetworkThroughput, i)
					}
				}
			}
		case "ProcessorArchitecture":
			p.ProcessorArchitecture = row[index]
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
		case "usageType":
			p.usageType = row[index]
		case "operation":
			p.operation = row[index]
		case "DedicatedEBSThroughput":
			p.DedicatedEBSThroughput = row[index]
		case "DeploymentModel":
			p.DeploymentModel = row[index]
		case "EngineMediaType":
			p.EngineMediaType = row[index]
		case "EnhancedNetworkingSupported":
			p.EnhancedNetworkingSupported = row[index]
		case "InstanceTypeFamily":
			p.InstanceTypeFamily = row[index]
		case "NormalizationSizeFactor":
			p.NormalizationSizeFactor = row[index]
		case "PricingUnit":
			p.PricingUnit = row[index]
		case "ProcessorFeatures":
			p.ProcessorFeatures = row[index]
		case "RegionCode":
			p.RegionCode = row[index]
		case "serviceName":
			p.serviceName = row[index]
		}
	}
}
