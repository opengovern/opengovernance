package recommendation

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"strconv"
)

func (s *Service) GCPComputeInstanceRecommendation(
	instance entity.GcpComputeInstance,
	metrics map[string][]entity.Datapoint,
	preferences map[string]*string,
) (*entity.GcpComputeInstanceRightsizingRecommendation, error) {
	machine, err := s.gcpComputeMachineTypeRepo.Get(instance.MachineType)
	if err != nil {
		return nil, err
	}
	currentCost, err := s.costSvc.GetGCPComputeInstanceCost(instance)
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

			Cost: currentCost,
		},
	}

	if _, ok := metrics["cpuUtilization"]; !ok {
		return nil, fmt.Errorf("cpuUtilization metric not found")
	}
	if _, ok := metrics["memoryUtilization"]; !ok {
		return nil, fmt.Errorf("memoryUtilization metric not found")
	}
	cpuUsage := extractGCPUsage(metrics["cpuUtilization"])
	memoryUsage := extractGCPUsage(metrics["memoryUtilization"])

	result.CPU = cpuUsage
	result.Memory = memoryUsage

	vCPU := machine.GuestCpus
	cpuBreathingRoom := int64(0)
	if preferences["CPUBreathingRoom"] != nil {
		cpuBreathingRoom, _ = strconv.ParseInt(*preferences["CPUBreathingRoom"], 10, 64)
	}
	memoryBreathingRoom := int64(0)
	if preferences["MemoryBreathingRoom"] != nil {
		memoryBreathingRoom, _ = strconv.ParseInt(*preferences["MemoryBreathingRoom"], 10, 64)
	}
	neededCPU := float64(vCPU) * (getValueOrZero(cpuUsage.Avg) + float64(cpuBreathingRoom)) / 100.0
	neededMemory := 0.0
	if memoryUsage.Avg != nil {
		neededMemory = calculateHeadroom(*memoryUsage.Avg, memoryBreathingRoom)
	}

	pref := make(map[string]interface{}) //TODO

	suggestedMachineType, err := s.gcpComputeMachineTypeRepo.GetCheapestByCoreAndMemory(neededCPU, neededMemory, pref)
	if err != nil {
		return nil, err
	}

	instance.Zone = suggestedMachineType.Zone
	instance.MachineType = suggestedMachineType.Name
	suggestedCost, err := s.costSvc.GetGCPComputeInstanceCost(instance)
	if err != nil {
		return nil, err
	}

	if suggestedMachineType != nil {
		result.Recommended = &entity.RightsizingGcpComputeInstance{
			Zone:          suggestedMachineType.Zone,
			MachineType:   suggestedMachineType.Name,
			MachineFamily: suggestedMachineType.MachineFamily,
			CPU:           suggestedMachineType.GuestCpus,
			MemoryMb:      suggestedMachineType.MemoryMb,

			Cost: suggestedCost,
		}
	}

	return &result, nil
}
