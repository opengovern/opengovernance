package recommendation

import (
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
)

type Service struct {
	ec2InstanceRepo repo.EC2InstanceTypeRepo
}

type Recommendation struct {
	Description string
	NewInstance entity.EC2Instance
	NewVolumes  []entity.EC2Volume

	CurrentInstanceType *model.EC2InstanceType
	NewInstanceType     *model.EC2InstanceType

	AvgNetworkBandwidth string
	AvgCPUUsage         string
}

func New(ec2InstanceRepo repo.EC2InstanceTypeRepo) *Service {
	return &Service{
		ec2InstanceRepo: ec2InstanceRepo,
	}
}
