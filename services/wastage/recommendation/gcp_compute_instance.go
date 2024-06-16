package recommendation

import (
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
)

func (s *Service) GCPComputeInstanceRecommendation(
	instance entity.GcpComputeInstance,
	metrics map[string][]types2.Datapoint,
	preferences map[string]*string,
) (*entity.GcpComputeInstanceRightsizingRecommendation, error) {
	machine, err := s.gcpComputeMachineTypeRepo.Get(instance.MachineType)
	if err != nil {
		return nil, err
	}
	result := entity.GcpComputeInstanceRightsizingRecommendation{
		Current: entity.RightsizingGcpComputeInstance{
			Zone:          instance.Zone,
			MachineType:   instance.MachineType,
			MachineFamily: machine.MachineFamily,
			CPU:           machine.GuestCpus,
			MemoryMb:      machine.MemoryMb,

			Cost: 0, // TODO: Calculate cost
		},
	}

	return &result, nil
}
