package recommendation

import (
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
)

type Service struct {
	ec2InstanceRepo repo.EC2InstanceTypeRepo
	ebsVolumeRepo   repo.EBSVolumeTypeRepo
}

type Ec2InstanceRecommendation struct {
	Description string
	NewInstance entity.EC2Instance
	NewVolumes  []entity.EC2Volume

	CurrentInstanceType *model.EC2InstanceType
	NewInstanceType     *model.EC2InstanceType

	AvgNetworkBandwidth string
	AvgCPUUsage         string
}

type EbsVolumeRecommendation struct {
	Description string
	NewVolume   types.Volume

	CurrentSize                  int32
	NewSize                      int32
	CurrentProvisionedIOPS       *int32
	NewProvisionedIOPS           *int32
	CurrentProvisionedThroughput *int32
	NewProvisionedThroughput     *int32
	CurrentVolumeType            types.VolumeType
	NewVolumeType                types.VolumeType

	AvgIOPS       int32
	AvgThroughput int32
}

func New(ec2InstanceRepo repo.EC2InstanceTypeRepo, ebsVolumeRepo repo.EBSVolumeTypeRepo) *Service {
	return &Service{
		ec2InstanceRepo: ec2InstanceRepo,
		ebsVolumeRepo:   ebsVolumeRepo,
	}
}
