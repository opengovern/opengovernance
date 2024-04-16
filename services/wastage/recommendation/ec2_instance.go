package recommendation

import (
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/kaytu-io/kaytu-aws-describer/aws/model"
)

func (s *Service) EC2InstanceRecommendation(region string, instance model.EC2InstanceDescription, volumes []model.EC2VolumeDescription, metrics map[string][]types2.Datapoint) ([]Recommendation, error) {
	return nil, nil
}
