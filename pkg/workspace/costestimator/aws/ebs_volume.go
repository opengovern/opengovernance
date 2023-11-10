package aws

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
)

func EC2VolumeCostByResource(db *db.Database, request api.GetEC2VolumeCostRequest) (float64, error) {
	volumeDescription := request.Volume.Volume
	cost, err := calcEC2VolumeCost(db, request.RegionCode, string(volumeDescription.VolumeType),
		*volumeDescription.Size, *volumeDescription.Iops)
	if err != nil {
		return 0, err
	}
	return cost * costestimator.TimeInterval, nil
}

// calcEC2VolumeCost Calculates ec2 volume (ebs volume) cost for one hour
func calcEC2VolumeCost(db *db.Database, region string, volumeType string, volumeSize int32, iops int32) (float64, error) {
	var cost float64
	volumeCost, err := db.FindEC2InstanceStoragePrice(region, volumeType)
	if err != nil {
		return 0, err
	}
	cost += volumeCost.Price * float64(volumeSize)
	if volumeType == "io1" || volumeType == "io2" {
		iopsCost, err := db.FindEC2InstanceSystemOperationPrice(region, volumeType, "EBS:VolumeP-IOPS")
		if err != nil {
			return 0, err
		}
		cost += iopsCost.Price * float64(iops)
	}
	numberOfDays := costestimator.GetNumberOfDays()
	return (cost / (float64(numberOfDays))) / 24, nil
}
