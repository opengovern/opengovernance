package model

import "gorm.io/gorm"

type EC2InstanceType struct {
	gorm.Model

	InstanceType           string
	VCpu                   int64 `gorm:"cpu_net"`
	MemoryGB               int64 `gorm:"cpu_net"`
	NetworkMaxBandwidth    int64 `gorm:"cpu_net"`
	NetworkIsBandwidthUpTo bool  `gorm:"cpu_net"`
	TermType               string
	Region                 string
	OperatingSystem        string
	LicenseModel           string
	PricePerUnit           float64
	Unit                   string
	UsageType              string
	PreInstalledSW         string
	Tenancy                string
	CapacityStatus         string
}
