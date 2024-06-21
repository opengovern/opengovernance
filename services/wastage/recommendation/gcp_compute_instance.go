package recommendation

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation/preferences/gcp_compute"
	"regexp"
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

	region := strings.Join([]string{strings.Split(instance.Zone, "-")[0], strings.Split(instance.Zone, "-")[1]}, "-")

	result := entity.GcpComputeInstanceRightsizingRecommendation{
		Current: entity.RightsizingGcpComputeInstance{
			Zone:          instance.Zone,
			Region:        region,
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
			vl = extractFromGCPComputeInstance(region, machine, k)
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
		if k == "MachineFamily" {
			if vl == "custom" {
				continue // TODO
			}
		}
		pref[fmt.Sprintf("%s %s ?", gcp_compute.PreferenceInstanceKey[k], cond)] = vl
	}

	suggestedMachineType, err := s.gcpComputeMachineTypeRepo.GetCheapestByCoreAndMemory(neededCPU, neededMemory, pref)
	if err != nil {
		return nil, err
	}

	excludeCustom := false
	if preferences["ExcludeCustomInstances"] != nil {
		if *preferences["ExcludeCustomInstances"] == "Yes" {
			excludeCustom = true
		}
	}

	if suggestedMachineType != nil {
		instance.Zone = suggestedMachineType.Zone
		instance.MachineType = suggestedMachineType.Name
		suggestedCost, err := s.costSvc.GetGCPComputeInstanceCost(instance)
		if err != nil {
			return nil, err
		}

		if !excludeCustom {
			customMachines, err := s.checkCustomMachines(region, neededCPU, neededMemory, preferences)
			if err != nil {
				return nil, err
			}
			for _, customMachine := range customMachines {
				if customMachine.cost < suggestedCost {
					suggestedMachineType = &customMachine.machineType
					suggestedCost = customMachine.cost
				}
			}
		}

		result.Recommended = &entity.RightsizingGcpComputeInstance{
			Zone:          suggestedMachineType.Zone,
			Region:        suggestedMachineType.Region,
			MachineType:   suggestedMachineType.Name,
			MachineFamily: suggestedMachineType.MachineFamily,
			CPU:           suggestedMachineType.GuestCpus,
			MemoryMb:      suggestedMachineType.MemoryMb,

			Cost: suggestedCost,
		}
	} else if !excludeCustom {
		customMachines, err := s.checkCustomMachines(region, neededCPU, neededMemory, preferences)
		if err != nil {
			return nil, err
		}
		suggestedMachineType = machine
		suggestedCost := currentCost

		for _, customMachine := range customMachines {
			if customMachine.cost < suggestedCost {
				suggestedMachineType = &customMachine.machineType
				suggestedCost = customMachine.cost
			}
		}

		result.Recommended = &entity.RightsizingGcpComputeInstance{
			Zone:          suggestedMachineType.Zone,
			Region:        suggestedMachineType.Region,
			MachineType:   suggestedMachineType.Name,
			MachineFamily: suggestedMachineType.MachineFamily,
			CPU:           suggestedMachineType.GuestCpus,
			MemoryMb:      suggestedMachineType.MemoryMb,

			Cost: suggestedCost,
		}
	}

	return &result, nil
}

func extractFromGCPComputeInstance(region string, machine *model.GCPComputeMachineType, k string) any {
	switch k {
	case "Region":
		return region
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
	re := regexp.MustCompile(`(\D.+)-(\d+)-(\d.+)`)
	machineTypePrefix := re.ReplaceAllString(instance.MachineType, "$1")
	strCPUAmount := re.ReplaceAllString(instance.MachineType, "$2")
	strRAMAmount := re.ReplaceAllString(instance.MachineType, "$3")

	region := strings.Join([]string{strings.Split(instance.Zone, "-")[0], strings.Split(instance.Zone, "-")[1]}, "-")
	cpu, err := strconv.ParseInt(strCPUAmount, 10, 64)
	if err != nil {
		return nil, err
	}
	memoryMb, err := strconv.ParseInt(strRAMAmount, 10, 64)
	if err != nil {
		return nil, err
	}

	family := "custom"
	if machineTypePrefix != "custom" {
		family = strings.Split(machineTypePrefix, "-")[0]
	}

	if family == "e2" {
		return nil, fmt.Errorf("e2 instances are not supported")
	}

	return &model.GCPComputeMachineType{
		Name:          instance.MachineType,
		MachineType:   instance.MachineType,
		MachineFamily: family,
		GuestCpus:     cpu,
		MemoryMb:      memoryMb,
		Zone:          instance.Zone,
		Region:        region,
		Description:   "",
		ImageSpaceGb:  0,

		UnitPrice: 0,
	}, nil
}

func (s *Service) checkCustomMachines(region string, neededCpu, neededMemory float64, preferences map[string]*string) ([]customOffer, error) {
	if preferences["MemoryGB"] != nil && *preferences["MemoryGB"] != "" {
		neededMemory, _ = strconv.ParseFloat(*preferences["MemoryGB"], 64)
	}
	if preferences["vCPU"] != nil && *preferences["vCPU"] != "" {
		neededCpu, _ = strconv.ParseFloat(*preferences["vCPU"], 64)
	}

	offers := make([]customOffer, 0)
	if preferences["MachineFamily"] != nil && *preferences["MachineFamily"] != "" {
		offer, err := s.checkCustomMachineForFamily(region, *preferences["MachineFamily"], neededCpu, neededMemory, preferences)
		if err != nil {
			return nil, err
		}
		if offer == nil {
			return nil, fmt.Errorf("machine family does not have any custom machines")
		}
		return offer, nil
	}

	n2Offer, err := s.checkCustomMachineForFamily(region, "n2", neededCpu, neededMemory, preferences)
	if err != nil {
		return nil, err
	}
	n4Offer, err := s.checkCustomMachineForFamily(region, "n4", neededCpu, neededMemory, preferences)
	if err != nil {
		return nil, err
	}
	n2dOffer, err := s.checkCustomMachineForFamily(region, "n2d", neededCpu, neededMemory, preferences)
	if err != nil {
		return nil, err
	}
	g2Offer, err := s.checkCustomMachineForFamily(region, "g2", neededCpu, neededMemory, preferences)
	if err != nil {
		return nil, err
	}
	offers = append(offers, n2Offer...)
	offers = append(offers, n4Offer...)
	offers = append(offers, n2dOffer...)
	offers = append(offers, g2Offer...)

	return offers, nil
}

func (s *Service) checkCustomMachineForFamily(region, family string, neededCpu, neededMemory float64, preferences map[string]*string) ([]customOffer, error) {
	pref := make(map[string]any)
	if preferences["Region"] != nil && *preferences["Region"] != "" {
		pref[fmt.Sprintf("%s %s ?", gcp_compute.PreferenceInstanceKey["Region"], "=")] = *preferences["Region"]
	} else if preferences["Region"] == nil {
		pref["location = ?"] = region
	}

	var customOffers []customOffer
	cpuSku, err := s.gcpComputeSKURepo.GetCheapestCustomCore(family, pref)
	if err != nil {
		return nil, err
	}
	if cpuSku == nil {
		return nil, nil
	}
	memorySku, err := s.gcpComputeSKURepo.GetCheapestCustomRam(family, pref)
	if err != nil {
		return nil, err
	}
	if memorySku == nil {
		return nil, nil
	}

	machineType := fmt.Sprintf("%s-custom-%d-%d", family, int64(neededCpu), int64(neededMemory))

	if memorySku.Location == cpuSku.Location {
		cost, err := s.costSvc.GetGCPComputeInstanceCost(entity.GcpComputeInstance{
			HashedInstanceId: "",
			Zone:             cpuSku.Location + "-a",
			MachineType:      machineType,
		})
		if err != nil {
			return nil, err
		}

		return []customOffer{{
			family: family,
			machineType: model.GCPComputeMachineType{
				Name:        machineType,
				MachineType: machineType,
				GuestCpus:   int64(neededCpu),
				MemoryMb:    int64(neededMemory),
				Zone:        cpuSku.Location + "-a",
				Region:      cpuSku.Location,
			},
			cost: cost,
		}}, nil
	}

	cpuRegionCost, err := s.costSvc.GetGCPComputeInstanceCost(entity.GcpComputeInstance{
		HashedInstanceId: "",
		Zone:             cpuSku.Location + "-a",
		MachineType:      machineType,
	})
	if err != nil {
		return nil, err
	}

	customOffers = append(customOffers, customOffer{
		family: family,
		machineType: model.GCPComputeMachineType{
			Name:        machineType,
			MachineType: machineType,
			GuestCpus:   int64(neededCpu),
			MemoryMb:    int64(neededMemory),
			Zone:        cpuSku.Location + "-a",
			Region:      cpuSku.Location,
		},
		cost: cpuRegionCost,
	})

	memoryRegionCost, err := s.costSvc.GetGCPComputeInstanceCost(entity.GcpComputeInstance{
		HashedInstanceId: "",
		Zone:             memorySku.Location + "-a",
		MachineType:      machineType,
	})
	if err != nil {
		return nil, err
	}

	customOffers = append(customOffers, customOffer{
		family: family,
		machineType: model.GCPComputeMachineType{
			Name:        machineType,
			MachineType: machineType,
			GuestCpus:   int64(neededCpu),
			MemoryMb:    int64(neededMemory),
			Zone:        memorySku.Location + "-a",
			Region:      memorySku.Location,
		},
		cost: memoryRegionCost,
	})

	return customOffers, nil
}

type customOffer struct {
	family      string
	machineType model.GCPComputeMachineType
	cost        float64
}
