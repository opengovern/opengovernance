package recommendation

import (
	"github.com/kaytu-io/kaytu-aws-describer/aws/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
)

type Service struct {
	ec2InstanceRepo repo.EC2InstanceTypeRepo
}

type Recommendation struct {
	Description string
	NewInstance model.EC2InstanceDescription
	NewVolumes  []model.EC2VolumeDescription
}

func New(ec2InstanceRepo repo.EC2InstanceTypeRepo) *Service {
	return &Service{
		ec2InstanceRepo: ec2InstanceRepo,
	}
}
