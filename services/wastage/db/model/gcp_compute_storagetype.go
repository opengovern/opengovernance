package model

import (
	"google.golang.org/api/compute/v1"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

type GCPComputeDiskType struct {
	gorm.Model

	// Basic fields
	Name        string `gorm:"index"`
	StorageType string `gorm:"index"`
	Zone        string `gorm:"index"`

	MinSizeGb int64
	MaxSizeGb int64
	Region    string

	UnitPrice float64
}

func (p *GCPComputeDiskType) PopulateFromObject(diskType *compute.DiskType) {
	p.Name = diskType.Name
	p.StorageType = diskType.Name

	diskZoneURLParts := strings.Split(diskType.Zone, "/")
	diskZone := diskZoneURLParts[len(diskZoneURLParts)-1]
	region := strings.Join([]string{strings.Split(diskZone, "-")[0], strings.Split(diskZone, "-")[1]}, "-")
	p.Region = region
	p.Zone = diskZone

	vds := strings.Split(diskType.ValidDiskSize, "-")
	minSizeGbStr, _ := strings.CutSuffix(vds[0], "GB")
	minSizeGb, _ := strconv.ParseInt(minSizeGbStr, 10, 64)
	p.MinSizeGb = minSizeGb

	maxSizeGbStr, _ := strings.CutSuffix(vds[1], "GB")
	maxSizeGb, _ := strconv.ParseInt(maxSizeGbStr, 10, 64)
	p.MaxSizeGb = maxSizeGb
}
