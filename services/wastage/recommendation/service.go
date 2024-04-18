package recommendation

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
)

type Service struct {
	ec2InstanceRepo repo.EC2InstanceTypeRepo
}

type Recommendation struct {
	Description     string
	NewInstance     types.Instance
	NewVolumes      []types.Volume
	NewInstanceType *model.EC2InstanceType
}

func New(ec2InstanceRepo repo.EC2InstanceTypeRepo) *Service {
	return &Service{
		ec2InstanceRepo: ec2InstanceRepo,
	}
}
