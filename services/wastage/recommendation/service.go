package recommendation

import (
	"github.com/kaytu-io/kaytu-engine/services/wastage/cost"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

type Service struct {
	logger *zap.Logger

	ec2InstanceRepo      repo.EC2InstanceTypeRepo
	ebsVolumeRepo        repo.EBSVolumeTypeRepo
	awsRDSDBInstanceRepo repo.RDSDBInstanceRepo
	awsRDSDBStorageRepo  repo.RDSDBStorageRepo
	openaiSvc            *openai.Client
	costSvc              *cost.Service
}

func New(logger *zap.Logger, ec2InstanceRepo repo.EC2InstanceTypeRepo, ebsVolumeRepo repo.EBSVolumeTypeRepo, awsRDSDBInstanceRepo repo.RDSDBInstanceRepo, awsRDSDBStorageRepo repo.RDSDBStorageRepo, token string, costSvc *cost.Service) *Service {
	return &Service{
		logger:               logger,
		ec2InstanceRepo:      ec2InstanceRepo,
		ebsVolumeRepo:        ebsVolumeRepo,
		awsRDSDBInstanceRepo: awsRDSDBInstanceRepo,
		awsRDSDBStorageRepo:  awsRDSDBStorageRepo,
		openaiSvc:            openai.NewClient(token),
		costSvc:              costSvc,
	}
}
