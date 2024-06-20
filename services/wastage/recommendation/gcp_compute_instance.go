package recommendation

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation/preferences/aws_rds"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation/preferences/gcp_compute"
	"strconv"
	"strings"
)

func (s *Service) GCPComputeInstanceRecommendation(
	instance entity.GcpComputeInstance,
	metrics map[string][]entity.Datapoint,
	preferences map[string]*string,
) (*entity.GcpComputeInstanceRightsizingRecommendation, error) {
	var machine *model.GCPComputeMachineType
	var err error

	if instance.MachineType == "" {
		return nil, fmt.Errorf("no machine type provided")
	}
	if strings.Contains(instance.MachineType, "custom") {
		machine, err = s.extractCustomInstanceDetails(instance)
	} else {
		machine, err = s.gcpComputeMachineTypeRepo.Get(instance.MachineType)
		if err != nil {
			return nil, err
		}
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
	neededCPU := float64(vCPU) * (getValueOrZero(cpuUsage.Avg) + (float64(cpuBreathingRoom) / 100.0))
	neededMemory := 0.0
	if memoryUsage.Avg != nil {
		neededMemory = calculateHeadroom(*memoryUsage.Avg/(1024*1024), memoryBreathingRoom)
	}

	pref := make(map[string]any)

	for k, v := range preferences {
		var vl any
		if v == nil {
			vl = extractFromGCPComputeInstance(instance, machine, k)
		} else {
			vl = *v
		}
		if _, ok := gcp_compute.PreferenceInstanceKey[k]; !ok {
			continue
		}

		cond := "="
		if sc, ok := gcp_compute.PreferenceInstanceSpecialCond[k]; ok {
			cond = sc
		}
		if k == "MemoryGB" {
			vl = int64(vl.(float64) * 1024)
		}
		pref[fmt.Sprintf("%s %s ?", aws_rds.PreferenceInstanceDBKey[k], cond)] = vl
	}

	suggestedMachineType, err := s.gcpComputeMachineTypeRepo.GetCheapestByCoreAndMemory(neededCPU, neededMemory, pref)
	if err != nil {
		return nil, err
	}

	if suggestedMachineType != nil {
		instance.Zone = suggestedMachineType.Zone
		instance.MachineType = suggestedMachineType.Name
		suggestedCost, err := s.costSvc.GetGCPComputeInstanceCost(instance)
		if err != nil {
			return nil, err
		}

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

func extractFromGCPComputeInstance(instance entity.GcpComputeInstance, machine *model.GCPComputeMachineType, k string) any {
	switch k {
	case "Region":
		return machine.Region
	case "Zone":
		return instance.Zone
	case "vCPU":
		return machine.GuestCpus
	case "MemoryGB":
		return machine.MemoryMb / 1024
	case "MachineFamily":
		return machine.MachineFamily
	case "MachineType":
		return machine.MachineType
	}
	return ""
}

func (s *Service) extractCustomInstanceDetails(instance entity.GcpComputeInstance) (*model.GCPComputeMachineType, error) {
	details := strings.Split(instance.MachineType, "-")
	if len(details) != 4 {
		return nil, fmt.Errorf("invalid custom instance type")
	}
	if details[0] == "e2" {
		return nil, fmt.Errorf("e2 instances are not supported")
	}
	region := strings.Join([]string{strings.Split(instance.Zone, "-")[0], strings.Split(instance.Zone, "-")[1]}, "-")
	cpu, err := strconv.ParseInt(details[2], 10, 64)
	if err != nil {
		return nil, err
	}
	memoryMb, err := strconv.ParseInt(details[3], 10, 64)
	if err != nil {
		return nil, err
	}

	return &model.GCPComputeMachineType{
		Name:          instance.MachineType,
		MachineType:   instance.MachineType,
		MachineFamily: details[0],
		GuestCpus:     cpu,
		MemoryMb:      memoryMb,
		Zone:          instance.Zone,
		Region:        region,
		Description:   "",
		ImageSpaceGb:  0,

		UnitPrice: 0,
	}, nil
}
