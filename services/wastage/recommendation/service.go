package recommendation

import (
	"github.com/kaytu-io/kaytu-engine/services/wastage/cost"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/sashabaranov/go-openai"
)

type Service struct {
	ec2InstanceRepo repo.EC2InstanceTypeRepo
	ebsVolumeRepo   repo.EBSVolumeTypeRepo
	openaiSvc       *openai.Client
	costSvc         *cost.Service
}

func New(ec2InstanceRepo repo.EC2InstanceTypeRepo, ebsVolumeRepo repo.EBSVolumeTypeRepo, token string, costSvc *cost.Service) *Service {

	return &Service{
		ec2InstanceRepo: ec2InstanceRepo,
		ebsVolumeRepo:   ebsVolumeRepo,
		openaiSvc:       openai.NewClient(token),
		costSvc:         costSvc,
	}
}
