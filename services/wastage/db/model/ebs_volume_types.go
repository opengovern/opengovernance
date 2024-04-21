package model

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

type EBSVolumeChargeType string

const (
	ChargeTypeIOPS       EBSVolumeChargeType = "IOPS"
	ChargeTypeSize       EBSVolumeChargeType = "Size"
	ChargeTypeThroughput EBSVolumeChargeType = "Throughput"
)

const (
	Io2ProvisionedIopsTier1UpperBound = 32000
	Io2ProvisionedIopsTier2UpperBound = 64000
	Gp3BaseThroughput                 = 125
	Gp3BaseIops                       = 3000
)

type EBSVolumeType struct {
	gorm.Model

	VolumeType   types.VolumeType
	ChargeType   EBSVolumeChargeType
	PricePerUnit float64
	PriceGroup   string

	MaxIops       int32
	MaxThroughput int32
	MaxSize       int

	TermType   string
	RegionCode string
}

func (v *EBSVolumeType) PopulateFromMap(columns map[string]int, row []string) {
	for col, index := range columns {
		switch col {
		case "Volume API Name":
			switch row[index] {
			case "gp2":
				v.VolumeType = types.VolumeTypeGp2
			case "gp3":
				v.VolumeType = types.VolumeTypeGp3
			case "io1":
				v.VolumeType = types.VolumeTypeIo1
			case "io2":
				v.VolumeType = types.VolumeTypeIo2
			case "sc1":
				v.VolumeType = types.VolumeTypeSc1
			case "st1":
				v.VolumeType = types.VolumeTypeSt1
			case "standard":
				v.VolumeType = types.VolumeTypeStandard
			}
		case "PricePerUnit":
			v.PricePerUnit, _ = strconv.ParseFloat(row[index], 64)
		case "Region Code":
			v.RegionCode = row[index]
		case "TermType":
			v.TermType = row[index]
		case "Group":
			v.PriceGroup = row[index]
		case "Product Family":
			switch row[index] {
			case "Storage":
				v.ChargeType = ChargeTypeSize
			case "Provisioned Throughput":
				v.ChargeType = ChargeTypeThroughput
			case "System Operation":
				v.ChargeType = ChargeTypeIOPS
			}
		case "Max throughput/volume":
			sections := strings.Split(row[index], " ")

			for _, numberSection := range sections {
				mt, err := strconv.ParseInt(numberSection, 10, 32)
				if err == nil {
					v.MaxThroughput = max(int32(mt), v.MaxThroughput)
				}
			}
		case "Max IOPS/volume":
			sections := strings.Split(row[index], " ")
			for _, numberSection := range sections {
				mi, err := strconv.ParseInt(numberSection, 10, 32)
				if err == nil {
					v.MaxIops = max(int32(mi), v.MaxIops)
				}
			}
		case "Max Volume Size":
			sections := strings.Split(row[index], " ")
			for _, numberSection := range sections {
				mv, err := strconv.ParseInt(numberSection, 10, 32)
				if err == nil {
					v.MaxSize = max(int(mv), v.MaxSize)
				}
			}
			// TiB to GiB
			v.MaxSize *= 1000
		}
	}
}
