package recommendation

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/sashabaranov/go-openai"
)

type Service struct {
	ec2InstanceRepo repo.EC2InstanceTypeRepo
	ebsVolumeRepo   repo.EBSVolumeTypeRepo
	openaiSvc       *openai.Client
}

type Ec2InstanceRecommendation struct {
	Description string
	NewInstance entity.EC2Instance
	NewVolumes  []entity.EC2Volume

	CurrentInstanceType *model.EC2InstanceType
	NewInstanceType     *model.EC2InstanceType

	AvgNetworkBandwidth      string
	AvgCPUUsage              string
	MaxMemoryUsagePercentage string
	AvgEBSBandwidth          string
}

type EbsVolumeRecommendation struct {
	Description string
	NewVolume   entity.EC2Volume

	CurrentSize                  int32
	NewSize                      int32
	CurrentProvisionedIOPS       *int32
	NewBaselineIOPS              *int32
	NewProvisionedIOPS           *int32
	CurrentProvisionedThroughput *float64
	NewBaselineThroughput        *float64
	NewProvisionedThroughput     *float64
	CurrentVolumeType            types.VolumeType
	NewVolumeType                types.VolumeType

	AvgIOPS       float64
	AvgThroughput float64
}

func New(ec2InstanceRepo repo.EC2InstanceTypeRepo, ebsVolumeRepo repo.EBSVolumeTypeRepo, token string) *Service {

	return &Service{
		ec2InstanceRepo: ec2InstanceRepo,
		ebsVolumeRepo:   ebsVolumeRepo,
		openaiSvc:       openai.NewClient(token),
	}
}
