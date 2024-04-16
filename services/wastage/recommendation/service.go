package recommendation

import (
	"github.com/kaytu-io/kaytu-aws-describer/aws/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
)

type Service struct {
	db *connector.Database
}

type Recommendation struct {
	Description string
	NewInstance model.EC2InstanceDescription
	NewVolumes  []model.EC2VolumeDescription
}

func New(db *connector.Database) *Service {
	return &Service{
		db: db,
	}
}
