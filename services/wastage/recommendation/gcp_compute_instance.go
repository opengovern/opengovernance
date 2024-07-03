package recommendation

import (
	"context"
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation/preferences/gcp_compute"
	gcp "github.com/kaytu-io/plugin-gcp/plugin/proto/src/golang/gcp"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"regexp"
	"strconv"
	"strings"
)

func (s *Service) GCPComputeInstanceRecommendation(
	ctx context.Context,
	instance gcp.GcpComputeInstance,
	metrics map[string]*gcp.Metric,
	preferences map[string]*wrapperspb.StringValue,
) (*gcp.GcpComputeInstanceRightsizingRecommendation, *model.GCPComputeMachineType, *model.GCPComputeMachineType, error) {
	var machine *model.GCPComputeMachineType
	var err error

	if instance.MachineType == "" {
		return nil, nil, nil, fmt.Errorf("no machine type provided")
	}
	if strings.Contains(instance.MachineType, "custom") {
		machine, err = s.extractCustomInstanceDetails(instance)
	} else {
		machine, err = s.gcpComputeMachineTypeRepo.Get(instance.MachineType)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	currentCost, err := s.costSvc.GetGCPComputeInstanceCost(ctx, instance)
	if err != nil {
		return nil, nil, nil, err
	}

	region := strings.Join([]string{strings.Split(instance.Zone, "-")[0], strings.Split(instance.Zone, "-")[1]}, "-")

	result := gcp.GcpComputeInstanceRightsizingRecommendation{
		Current: &gcp.RightsizingGcpComputeInstance{
			Zone:          instance.Zone,
			Region:        region,
			MachineType:   instance.MachineType,
			MachineFamily: machine.MachineFamily,
			Cpu:           machine.GuestCpus,
			MemoryMb:      machine.MemoryMb,

			Cost: currentCost,
		},
	}

	if v, ok := metrics["cpuUtilization"]; !ok || v == nil {
		return nil, nil, nil, fmt.Errorf("cpuUtilization metric not found")
	}
	if v, ok := metrics["memoryUtilization"]; !ok || v == nil {
		return nil, nil, nil, fmt.Errorf("memoryUtilization metric not found")
	}
	cpuUsage := extractGCPUsage(metrics["cpuUtilization"].Data)
	memoryUsage := extractGCPUsage(metrics["memoryUtilization"].Data)

	result.Cpu = &cpuUsage
	result.Memory = &memoryUsage

	vCPU := machine.GuestCpus
	cpuBreathingRoom := int64(0)
	if preferences["CPUBreathingRoom"] != nil {
		cpuBreathingRoom, _ = strconv.ParseInt(preferences["CPUBreathingRoom"].GetValue(), 10, 64)
	}
	memoryBreathingRoom := int64(0)
	if preferences["MemoryBreathingRoom"] != nil {
		memoryBreathingRoom, _ = strconv.ParseInt(preferences["MemoryBreathingRoom"].GetValue(), 10, 64)
	}
	neededCPU := float64(vCPU) * (PWrapperDouble(cpuUsage.Avg) + (float64(cpuBreathingRoom) / 100.0))
	if neededCPU < 2 {
		neededCPU = 2
	}

	neededMemoryMb := 0.0
	if memoryUsage.Avg != nil {
		neededMemoryMb = calculateHeadroom(PWrapperDouble(memoryUsage.Avg)/(1024*1024), memoryBreathingRoom)
	}
	if neededMemoryMb < 1024 {
		neededMemoryMb = 1024
	}

	pref := make(map[string]any)

	for k, v := range preferences {
		var vl any
		if v == nil {
			vl = extractFromGCPComputeInstance(region, machine, k)
		} else {
			vl = v.GetValue()
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
				continue
			}
		}
		pref[fmt.Sprintf("%s %s ?", gcp_compute.PreferenceInstanceKey[k], cond)] = vl
	}

	suggestedMachineType, err := s.gcpComputeMachineTypeRepo.GetCheapestByCoreAndMemory(neededCPU, neededMemoryMb, pref)
	if err != nil {
		return nil, nil, nil, err
	}

	excludeCustom := false
	if preferences["ExcludeCustomInstances"] != nil {
		if preferences["ExcludeCustomInstances"].GetValue() == "Yes" {
			excludeCustom = true
		}
	}

	if suggestedMachineType != nil {
		instance.Zone = suggestedMachineType.Zone
		instance.MachineType = suggestedMachineType.Name
		suggestedCost, err := s.costSvc.GetGCPComputeInstanceCost(ctx, instance)
		if err != nil {
			return nil, nil, nil, err
		}

		if !excludeCustom {
			customMachines, err := s.checkCustomMachines(ctx, region, int64(neededCPU), int64(neededMemoryMb), preferences)
			if err != nil {
				return nil, nil, nil, err
			}
			for _, customMachine := range customMachines {
				if customMachine.Cost < suggestedCost {
					suggestedMachineType = &customMachine.MachineType
					suggestedCost = customMachine.Cost
				}
			}
		}

		result.Recommended = &gcp.RightsizingGcpComputeInstance{
			Zone:          suggestedMachineType.Zone,
			Region:        suggestedMachineType.Region,
			MachineType:   suggestedMachineType.Name,
			MachineFamily: suggestedMachineType.MachineFamily,
			Cpu:           suggestedMachineType.GuestCpus,
			MemoryMb:      suggestedMachineType.MemoryMb,

			Cost: suggestedCost,
		}
	} else if !excludeCustom {
		customMachines, err := s.checkCustomMachines(ctx, region, int64(neededCPU), int64(neededMemoryMb), preferences)
		if err != nil {
			return nil, nil, nil, err
		}
		suggestedMachineType = machine
		suggestedCost := currentCost

		for _, customMachine := range customMachines {
			if customMachine.Cost < suggestedCost {
				suggestedMachineType = &customMachine.MachineType
				suggestedCost = customMachine.Cost
			}
		}

		result.Recommended = &gcp.RightsizingGcpComputeInstance{
			Zone:          suggestedMachineType.Zone,
			Region:        suggestedMachineType.Region,
			MachineType:   suggestedMachineType.Name,
			MachineFamily: suggestedMachineType.MachineFamily,
			Cpu:           suggestedMachineType.GuestCpus,
			MemoryMb:      suggestedMachineType.MemoryMb,

			Cost: suggestedCost,
		}
	}
	if suggestedMachineType == nil {
		suggestedMachineType = machine
	}

	description, err := s.generateGcpComputeInstanceDescription(region, instance, metrics, preferences, neededCPU,
		neededMemoryMb, machine, suggestedMachineType)
	if err != nil {
		s.logger.Error("Failed to generate description", zap.Error(err))
	} else {
		result.Description = description
	}

	if preferences["ExcludeUpsizingFeature"] != nil {
		if preferences["ExcludeUpsizingFeature"].GetValue() == "Yes" {
			if result.Recommended != nil && result.Recommended.Cost > result.Current.Cost {
				result.Recommended = result.Current
				result.Description = "No recommendation available as upsizing feature is disabled"
				return &result, machine, machine, nil
			}
		}
	}

	return &result, machine, suggestedMachineType, nil
}

func (s *Service) GCPComputeDiskRecommendation(
	ctx context.Context,
	disk gcp.GcpComputeDisk,
	currentMachine *model.GCPComputeMachineType,
	recommendedMachine *model.GCPComputeMachineType,
	metrics gcp.DiskMetrics,
	preferences map[string]*wrapperspb.StringValue,
) (*gcp.GcpComputeDiskRecommendation, error) {
	currentCost, err := s.costSvc.GetGCPComputeDiskCost(ctx, disk)
	if err != nil {
		return nil, err
	}

	readIopsUsage := extractGCPUsage(metrics.Metrics["DiskReadIOPS"].Data)
	writeIopsUsage := extractGCPUsage(metrics.Metrics["DiskWriteIOPS"].Data)
	readThroughputUsageBytes := extractGCPUsage(metrics.Metrics["DiskReadThroughput"].Data)
	readThroughputUsageMb := gcp.Usage{
		Avg: funcPWrapper(readThroughputUsageBytes.Avg, readThroughputUsageBytes.Avg, func(a, _ float64) float64 { return a / (1024 * 1024) }),
		Min: funcPWrapper(readThroughputUsageBytes.Min, readThroughputUsageBytes.Min, func(a, _ float64) float64 { return a / (1024 * 1024) }),
		Max: funcPWrapper(readThroughputUsageBytes.Max, readThroughputUsageBytes.Max, func(a, _ float64) float64 { return a / (1024 * 1024) }),
	}
	writeThroughputUsageBytes := extractGCPUsage(metrics.Metrics["DiskWriteThroughput"].Data)
	writeThroughputUsageMb := gcp.Usage{
		Avg: funcPWrapper(writeThroughputUsageBytes.Avg, writeThroughputUsageBytes.Avg, func(a, _ float64) float64 { return a / (1024 * 1024) }),
		Min: funcPWrapper(writeThroughputUsageBytes.Min, writeThroughputUsageBytes.Min, func(a, _ float64) float64 { return a / (1024 * 1024) }),
		Max: funcPWrapper(writeThroughputUsageBytes.Max, writeThroughputUsageBytes.Max, func(a, _ float64) float64 { return a / (1024 * 1024) }),
	}

	readIopsLimit, writeIopsLimit, readThroughputLimit, writeThroughputLimit, err := s.getMaximums(currentMachine.MachineFamily,
		currentMachine.MachineType, disk.DiskType, currentMachine.GuestCpus, disk.DiskSize.Value)
	if err != nil {
		return nil, err
	}

	result := gcp.GcpComputeDiskRecommendation{
		Current: &gcp.RightsizingGcpComputeDisk{
			DiskType:             disk.DiskType,
			DiskSize:             disk.DiskSize.Value,
			ReadIopsLimit:        readIopsLimit,
			WriteIopsLimit:       writeIopsLimit,
			ReadThroughputLimit:  readThroughputLimit,
			WriteThroughputLimit: writeThroughputLimit,

			Zone:   disk.Zone,
			Region: disk.Region,

			Cost: currentCost,
		},
		ReadIops:        &readIopsUsage,
		WriteIops:       &writeIopsUsage,
		ReadThroughput:  &readThroughputUsageMb,
		WriteThroughput: &writeThroughputUsageMb,
	}

	iopsBreathingRoom := int64(0)
	if preferences["IOPSBreathingRoom"] != nil {
		iopsBreathingRoom, _ = strconv.ParseInt(preferences["IopsBreathingRoom"].GetValue(), 10, 64)
	}

	throughputBreathingRoom := int64(0)
	if preferences["ThroughputBreathingRoom"] != nil {
		throughputBreathingRoom, _ = strconv.ParseInt(preferences["ThroughputBreathingRoom"].GetValue(), 10, 64)
	}

	neededReadIops := pWrapperCalculateHeadroom(readIopsUsage.Avg, iopsBreathingRoom)
	neededReadThroughput := pWrapperCalculateHeadroom(readThroughputUsageMb.Avg, throughputBreathingRoom)
	neededWriteIops := pWrapperCalculateHeadroom(writeIopsUsage.Avg, iopsBreathingRoom)
	neededWriteThroughput := pWrapperCalculateHeadroom(writeThroughputUsageMb.Avg, throughputBreathingRoom)

	pref := make(map[string]any)

	diskSize := disk.DiskSize.Value
	if ds, ok := preferences["DiskSizeGb"]; ok {
		if ds != nil {
			diskSize, _ = strconv.ParseInt(ds.GetValue(), 10, 64)
		}
	}

	suggestions, err := s.findCheapestDiskType(recommendedMachine.MachineFamily, recommendedMachine.MachineType, recommendedMachine.GuestCpus,
		neededReadIops, neededWriteIops, neededReadThroughput, neededWriteThroughput, diskSize)
	if err != nil {
		return nil, err
	}

	var suggestedType *string
	var suggestedSize *int64

	if suggestions != nil && len(suggestions) > 0 {
		for i, _ := range suggestions {
			newDisk := gcp.GcpComputeDisk{
				Id:       disk.Id,
				Zone:     disk.Zone,
				Region:   disk.Region,
				DiskType: suggestions[i].Type,
				DiskSize: wrapperspb.Int64(suggestions[i].Size),
			}
			suggestedCost, err := s.costSvc.GetGCPComputeDiskCost(ctx, newDisk)
			if err != nil {
				return nil, err
			}
			suggestions[i].Cost = &suggestedCost
		}
		s.logger.Info("Disk suggestions", zap.Any("suggestions", suggestions))
		minPriceSuggestion := suggestions[0]
		for _, sug := range suggestions {
			if _, ok := preferences["DiskSizeGb"]; ok {
				if diskSize != minPriceSuggestion.Size {
					continue
				}
			}
			if *sug.Cost < *minPriceSuggestion.Cost {
				minPriceSuggestion = sug
			}
		}
		suggestedType = &minPriceSuggestion.Type
		suggestedSize = &minPriceSuggestion.Size
	}

	if suggestedType == nil && suggestedSize == nil {
		suggestedType = &disk.DiskType
		suggestedSize = &disk.DiskSize.Value
	}

	pref["storage_type = ?"] = suggestedType

	for k, v := range preferences {
		var vl any
		if v == nil {
			vl = extractFromGCPComputeDisk(disk, k)
		} else {
			vl = v.GetValue()
		}
		if _, ok := gcp_compute.PreferenceDiskKey[k]; !ok {
			continue
		}

		cond := "="

		pref[fmt.Sprintf("%s %s ?", gcp_compute.PreferenceDiskKey[k], cond)] = vl
	}

	suggestedStorageType, err := s.gcpComputeDiskTypeRepo.GetCheapest(pref)
	if err != nil {
		return nil, err
	}
	recommendedReadIopsLimit, recommendedWriteIopsLimit, recommendedReadThroughputLimit, recommendedWriteThroughputLimit, err := s.getMaximums(recommendedMachine.MachineFamily,
		recommendedMachine.MachineType, suggestedStorageType.StorageType, recommendedMachine.GuestCpus, PWrapperInt64(disk.DiskSize))
	if err != nil {
		return nil, err
	}

	if suggestedStorageType != nil {
		disk.Zone = suggestedStorageType.Zone
		disk.DiskType = *suggestedType
		disk.Region = suggestedStorageType.Region
		disk.DiskSize = PInt64Wrapper(suggestedSize)
		suggestedCost, err := s.costSvc.GetGCPComputeDiskCost(ctx, disk)
		if err != nil {
			return nil, err
		}

		result.Recommended = &gcp.RightsizingGcpComputeDisk{
			Zone:                 suggestedStorageType.Zone,
			Region:               suggestedStorageType.Region,
			DiskType:             suggestedStorageType.StorageType,
			DiskSize:             disk.DiskSize.Value,
			ReadIopsLimit:        recommendedReadIopsLimit,
			WriteIopsLimit:       recommendedWriteIopsLimit,
			ReadThroughputLimit:  recommendedReadThroughputLimit,
			WriteThroughputLimit: recommendedWriteThroughputLimit,

			Cost: suggestedCost,
		}
	}

	description, err := s.generateGcpComputeDiskDescription(disk, currentMachine, recommendedMachine, metrics,
		preferences, readIopsLimit, writeIopsLimit, readThroughputLimit, writeThroughputLimit, neededReadIops,
		neededWriteIops, neededReadThroughput, neededWriteThroughput, recommendedReadIopsLimit, recommendedWriteIopsLimit,
		recommendedReadThroughputLimit, recommendedWriteThroughputLimit, *suggestedType, *suggestedSize)
	if err != nil {
		s.logger.Error("Failed to generate description", zap.Error(err))
	} else {
		result.Description = description
	}

	if preferences["ExcludeUpsizingFeature"] != nil {
		if preferences["ExcludeUpsizingFeature"].GetValue() == "Yes" {
			if result.Recommended != nil && result.Recommended.Cost > result.Current.Cost {
				result.Recommended = result.Current
				result.Description = "No recommendation available as upsizing feature is disabled"
				return &result, nil
			}
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

func extractFromGCPComputeDisk(disk gcp.GcpComputeDisk, k string) any {
	switch k {
	case "Region":
		return disk.Region
	case "DiskType":
		return disk.DiskType
	}
	return ""
}

func (s *Service) extractCustomInstanceDetails(instance gcp.GcpComputeInstance) (*model.GCPComputeMachineType, error) {
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

func (s *Service) checkCustomMachines(ctx context.Context, region string, neededCpu, neededMemoryMb int64, preferences map[string]*wrapperspb.StringValue) ([]CustomOffer, error) {
	if preferences["MemoryGB"] != nil && preferences["MemoryGB"].GetValue() != "" {
		neededMemoryGb, _ := strconv.ParseInt(preferences["MemoryGB"].GetValue(), 10, 64)
		neededMemoryMb = neededMemoryGb * 1024
	}
	if preferences["vCPU"] != nil && preferences["vCPU"].GetValue() != "" {
		neededCpu, _ = strconv.ParseInt(preferences["vCPU"].GetValue(), 10, 64)
	}

	offers := make([]CustomOffer, 0)
	if preferences["MachineFamily"] != nil && preferences["MachineFamily"].GetValue() != "" {
		offer, err := s.checkCustomMachineForFamily(ctx, region, preferences["MachineFamily"].GetValue(), neededCpu, neededMemoryMb, preferences)
		if err != nil {
			return nil, err
		}
		if offer == nil {
			return nil, fmt.Errorf("machine family does not have any custom machines")
		}
		return offer, nil
	}

	if neededCpu <= 128 && neededMemoryMb <= 665600 {
		n2Offer, err := s.checkCustomMachineForFamily(ctx, region, "n2", neededCpu, neededMemoryMb, preferences)
		if err != nil {
			return nil, err
		}
		offers = append(offers, n2Offer...)
	}
	if neededCpu <= 80 && neededMemoryMb <= 665600 {
		n4Offer, err := s.checkCustomMachineForFamily(ctx, region, "n4", neededCpu, neededMemoryMb, preferences)
		if err != nil {
			return nil, err
		}
		offers = append(offers, n4Offer...)
	}
	if neededCpu <= 224 && neededMemoryMb <= 786432 {
		n2dOffer, err := s.checkCustomMachineForFamily(ctx, region, "n2d", neededCpu, neededMemoryMb, preferences)
		if err != nil {
			return nil, err
		}
		offers = append(offers, n2dOffer...)
	}
	// TODO: add e2 custom machines
	g2Offer, err := s.checkCustomMachineForFamily(ctx, region, "g2", neededCpu, neededMemoryMb, preferences)
	if err != nil {
		return nil, err
	}
	offers = append(offers, g2Offer...)

	s.logger.Info("custom machines", zap.Any("offers", offers))
	for _, offer := range offers {
		s.logger.Info("custom machine info", zap.String("family", offer.Family), zap.Any("machineType", offer.MachineType), zap.Float64("cost", offer.Cost))
	}

	return offers, nil
}

func (s *Service) checkCustomMachineForFamily(ctx context.Context, region, family string, neededCpu, neededMemoryMb int64, preferences map[string]*wrapperspb.StringValue) ([]CustomOffer, error) {
	if neededCpu > 2 {
		neededCpu = roundUpToMultipleOf(neededCpu, 4)
	}
	if family == "n2" || family == "n2d" {
		neededMemoryMb = roundUpToMultipleOf(neededMemoryMb, 256)
		if neededMemoryMb < neededCpu*512 {
			neededMemoryMb = neededCpu * 512
		}
	} else if family == "n4" {
		neededMemoryMb = roundUpToMultipleOf(neededMemoryMb, 256)
		if neededMemoryMb < neededCpu*2048 {
			neededMemoryMb = neededCpu * 2048
		}
	} else if family == "g2" {
		neededMemoryMb = roundUpToMultipleOf(neededMemoryMb, 1024)
		if neededMemoryMb < neededCpu*4096 {
			neededMemoryMb = neededCpu * 4096
		}
	}

	if neededMemoryMb > 8192*neededCpu {
		neededCpu = roundUpToMultipleOf(neededMemoryMb, 8192) / 8192
		neededCpu = roundUpToMultipleOf(neededCpu, 4)
	}

	pref := make(map[string]any)
	for k, v := range preferences {
		if k == "Region" {
			if v != nil && v.GetValue() != "" {
				pref["location = ?"] = *v
			} else {
				pref["location = ?"] = region
			}
		}
	}

	var customOffers []CustomOffer
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

	machineType := fmt.Sprintf("%s-custom-%d-%d", family, neededCpu, neededMemoryMb)

	if memorySku.Location == cpuSku.Location {
		cost, err := s.costSvc.GetGCPComputeInstanceCost(ctx, gcp.GcpComputeInstance{
			Id:          "",
			Zone:        cpuSku.Location + "-a",
			MachineType: machineType,
		})
		if err != nil {
			return nil, err
		}

		return []CustomOffer{{
			Family: family,
			MachineType: model.GCPComputeMachineType{
				Name:        machineType,
				MachineType: machineType,
				GuestCpus:   neededCpu,
				MemoryMb:    neededMemoryMb,
				Zone:        cpuSku.Location + "-a",
				Region:      cpuSku.Location,
			},
			Cost: cost,
		}}, nil
	}

	cpuRegionCost, err := s.costSvc.GetGCPComputeInstanceCost(ctx, gcp.GcpComputeInstance{
		Id:          "",
		Zone:        cpuSku.Location + "-a",
		MachineType: machineType,
	})
	if err != nil {
		return nil, err
	}

	customOffers = append(customOffers, CustomOffer{
		Family: family,
		MachineType: model.GCPComputeMachineType{
			Name:        machineType,
			MachineType: machineType,
			GuestCpus:   neededCpu,
			MemoryMb:    neededMemoryMb,
			Zone:        cpuSku.Location + "-a",
			Region:      cpuSku.Location,
		},
		Cost: cpuRegionCost,
	})

	memoryRegionCost, err := s.costSvc.GetGCPComputeInstanceCost(ctx, gcp.GcpComputeInstance{
		Id:          "",
		Zone:        memorySku.Location + "-a",
		MachineType: machineType,
	})
	if err != nil {
		return nil, err
	}

	customOffers = append(customOffers, CustomOffer{
		Family: family,
		MachineType: model.GCPComputeMachineType{
			Name:        machineType,
			MachineType: machineType,
			GuestCpus:   neededCpu,
			MemoryMb:    neededMemoryMb,
			Zone:        memorySku.Location + "-a",
			Region:      memorySku.Location,
		},
		Cost: memoryRegionCost,
	})

	return customOffers, nil
}

type CustomOffer struct {
	Family      string
	MachineType model.GCPComputeMachineType
	Cost        float64
}

func roundUpToMultipleOf(number, multipleOf int64) int64 {
	if number%multipleOf == 0 {
		return number
	}
	return ((number / multipleOf) + 1) * multipleOf
}

func (s *Service) generateGcpComputeInstanceDescription(region string, instance gcp.GcpComputeInstance,
	metrics map[string]*gcp.Metric, preferences map[string]*wrapperspb.StringValue,
	neededCpu, neededMemoryMb float64, currentMachine *model.GCPComputeMachineType,
	suggestedMachineType *model.GCPComputeMachineType) (string, error) {
	if v, ok := metrics["cpuUtilization"]; !ok || v == nil {
		return "", fmt.Errorf("cpuUtilization metric not found")
	}
	if v, ok := metrics["memoryUtilization"]; !ok || v == nil {
		return "", fmt.Errorf("memoryUtilization metric not found")
	}
	cpuUsage := extractGCPUsage(metrics["cpuUtilization"].Data)
	memoryUsage := extractGCPUsage(metrics["memoryUtilization"].Data)

	var usage string
	if len(metrics["cpuUtilization"].Data) > 0 {
		usage = fmt.Sprintf("- %s has %d vCPUs. Usage over the course of last week is min=%.2f%%, avg=%.2f%%, max=%.2f%%, so you only need %.2f vCPUs. %s has %d vCPUs.\n", instance.MachineType, currentMachine.GuestCpus, PWrapperDouble(cpuUsage.Min), PWrapperDouble(cpuUsage.Avg), PWrapperDouble(cpuUsage.Max), neededCpu, suggestedMachineType.MachineType, suggestedMachineType.GuestCpus)
	} else {
		usage = fmt.Sprintf("- %s has %d vCPUs. Usage is not available. You need to install CloudWatch Agent on your instance to get this data. %s has %d vCPUs.\n", instance.MachineType, currentMachine.GuestCpus, suggestedMachineType.MachineType, suggestedMachineType.GuestCpus)

	}
	if len(metrics["memoryUtilization"].Data) > 0 {
		usage += fmt.Sprintf("- %s has %dMb Memory. Usage over the course of last week is min=%.2f%%, avg=%.2f%%, max=%.2f%%, so you only need %.2fMb Memory. %s has %dMb Memory.\n", instance.MachineType, currentMachine.MemoryMb, PWrapperDouble(memoryUsage.Min), PWrapperDouble(memoryUsage.Avg), PWrapperDouble(memoryUsage.Max), neededMemoryMb, suggestedMachineType.MachineType, suggestedMachineType.MemoryMb)
	} else {
		usage += fmt.Sprintf("- %s has %dMb Memory. Usage is not available. You need to install CloudWatch Agent on your instance to get this data. %s has %dMb Memory.\n", instance.MachineType, currentMachine.MemoryMb, suggestedMachineType.MachineType, suggestedMachineType.MemoryMb)
	}

	needs := ""
	for k, v := range preferences {
		if gcp_compute.PreferenceInstanceKey[k] == "" {
			continue
		}
		if v == nil {
			vl := extractFromGCPComputeInstance(region, currentMachine, k)
			needs += fmt.Sprintf("- You asked %s to be same as the current instance value which is %v\n", k, vl)
		} else {
			needs += fmt.Sprintf("- You asked %s to be %s\n", k, *v)
		}
	}

	prompt := fmt.Sprintf(`
I'm giving recommendation on GCP Compute Instance right sizing. Based on user's usage and needs I have concluded that the best option for him is to use %s instead of %s. I need help summarizing the explanation into 280 characters (it's not a tweet! dont use hashtag!) while keeping these rules:
- mention the requirements from user side.
- for those fields which are changing make sure you mention the change.

Here's usage data:
%s

User's needs:
%s
`, suggestedMachineType.MachineType, currentMachine.MachineType, usage, needs)

	resp, err := s.openaiSvc.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4TurboPreview,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("empty choices")
	}

	s.logger.Info("GPT results", zap.String("prompt", prompt), zap.String("result", strings.TrimSpace(resp.Choices[0].Message.Content)))

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func (s *Service) generateGcpComputeDiskDescription(disk gcp.GcpComputeDisk,
	currentMachine *model.GCPComputeMachineType,
	recommendedMachine *model.GCPComputeMachineType,
	metrics gcp.DiskMetrics, preferences map[string]*wrapperspb.StringValue,
	readIopsLimit, writeIopsLimit int64, readThroughputLimit, writeThroughputLimit float64,
	neededReadIops, neededWriteIops, neededReadThroughput, neededWriteThroughput float64,
	recommendedReadIopsLimit, recommendedWriteIopsLimit int64, recommendedReadThroughputLimit, recommendedWriteThroughputLimit float64,
	suggestedType string, suggestedSize int64,
) (string, error) {
	if v, ok := metrics.Metrics["DiskReadIOPS"]; !ok || v == nil {
		return "", fmt.Errorf("DiskReadIOPS metric not found")
	}
	if v, ok := metrics.Metrics["DiskWriteIOPS"]; !ok || v == nil {
		return "", fmt.Errorf("DiskWriteIOPS metric not found")
	}
	if v, ok := metrics.Metrics["DiskReadThroughput"]; !ok || v == nil {
		return "", fmt.Errorf("DiskReadThroughput metric not found")
	}
	if v, ok := metrics.Metrics["DiskWriteThroughput"]; !ok || v == nil {
		return "", fmt.Errorf("DiskWriteThroughput metric not found")
	}
	readIopsUsage := extractGCPUsage(metrics.Metrics["DiskReadIOPS"].Data)
	writeIopsUsage := extractGCPUsage(metrics.Metrics["DiskWriteIOPS"].Data)
	readThroughputUsageBytes := extractGCPUsage(metrics.Metrics["DiskReadThroughput"].Data)
	readThroughputUsageMb := gcp.Usage{
		Avg: funcPWrapper(readThroughputUsageBytes.Avg, readThroughputUsageBytes.Avg, func(a, _ float64) float64 { return a / (1024 * 1024) }),
		Min: funcPWrapper(readThroughputUsageBytes.Min, readThroughputUsageBytes.Min, func(a, _ float64) float64 { return a / (1024 * 1024) }),
		Max: funcPWrapper(readThroughputUsageBytes.Max, readThroughputUsageBytes.Max, func(a, _ float64) float64 { return a / (1024 * 1024) }),
	}
	writeThroughputUsageBytes := extractGCPUsage(metrics.Metrics["DiskWriteThroughput"].Data)
	writeThroughputUsageMb := gcp.Usage{
		Avg: funcPWrapper(writeThroughputUsageBytes.Avg, writeThroughputUsageBytes.Avg, func(a, _ float64) float64 { return a / (1024 * 1024) }),
		Min: funcPWrapper(writeThroughputUsageBytes.Min, writeThroughputUsageBytes.Min, func(a, _ float64) float64 { return a / (1024 * 1024) }),
		Max: funcPWrapper(writeThroughputUsageBytes.Max, writeThroughputUsageBytes.Max, func(a, _ float64) float64 { return a / (1024 * 1024) }),
	}

	var usage string
	if len(metrics.Metrics["DiskReadIOPS"].Data) > 0 || len(metrics.Metrics["DiskWriteIOPS"].Data) > 0 {
		usage = fmt.Sprintf("- Disk Type %s with Machine Type %s with size %d has %d Write IOPS estimation. Usage over the course of last week is min=%.2f%%, avg=%.2f%%, max=%.2f%%, so you only need %.1f Write IOPS estimation. Disk Type %s with Machine Type %s with size %d has %d Write IOPS estimation.\n", disk.DiskType, currentMachine.MachineType, PWrapperInt64(disk.DiskSize), writeIopsLimit, PWrapperDouble(writeIopsUsage.Min), PWrapperDouble(writeIopsUsage.Avg), PWrapperDouble(writeIopsUsage.Max), neededWriteIops, suggestedType, recommendedMachine.MachineType, suggestedSize, recommendedWriteIopsLimit)
		usage += fmt.Sprintf("- Disk Type %s with Machine Type %s with size %d has %d Read IOPS estimation. Usage over the course of last week is min=%.2f%%, avg=%.2f%%, max=%.2f%%, so you only need %.1f Read IOPS estimation. Disk Type %s with Machine Type %s with size %d has %d Read IOPS estimation.\n", disk.DiskType, currentMachine.MachineType, PWrapperInt64(disk.DiskSize), readIopsLimit, PWrapperDouble(readIopsUsage.Min), PWrapperDouble(readIopsUsage.Avg), PWrapperDouble(readIopsUsage.Max), neededReadIops, suggestedType, recommendedMachine.MachineType, suggestedSize, recommendedReadIopsLimit)
	} else {
		usage = fmt.Sprintf("- Disk Type %s with Machine Type %s with size %d has %d Write IOPS estimation. Usage is not available. You need to install CloudWatch Agent on your instance to get this data. Disk Type %s with Machine Type %s with size %d has %d IOPS estimation.\n", disk.DiskType, currentMachine.MachineType, PWrapperInt64(disk.DiskSize), writeIopsLimit, suggestedType, recommendedMachine.MachineType, suggestedSize, recommendedWriteIopsLimit)
		usage += fmt.Sprintf("- Disk Type %s with Machine Type %s with size %d has %d Write IOPS estimation. Usage is not available. You need to install CloudWatch Agent on your instance to get this data. Disk Type %s with Machine Type %s with size %d has %d IOPS estimation.\n", disk.DiskType, currentMachine.MachineType, PWrapperInt64(disk.DiskSize), readIopsLimit, suggestedType, recommendedMachine.MachineType, suggestedSize, recommendedReadIopsLimit)
	}
	if len(metrics.Metrics["DiskReadThroughput"].Data) > 0 || len(metrics.Metrics["DiskWriteThroughput"].Data) > 0 {
		usage += fmt.Sprintf("- Disk Type %s with Machine Type %s with size %d has %.2f Mb Write Throughput estimation. Usage over the course of last week is min=%.2f%%, avg=%.2f%%, max=%.2f%%, so you only need %.2f Mb Write Throughput estimation. Disk Type %s with Machine Type %s with size %d has %.2f Mb Write Throughput estimation.\n", disk.DiskType, currentMachine.MachineType, PWrapperInt64(disk.DiskSize), writeThroughputLimit, PWrapperDouble(writeThroughputUsageMb.Min), PWrapperDouble(writeThroughputUsageMb.Avg), PWrapperDouble(writeThroughputUsageMb.Max), neededWriteThroughput, suggestedType, recommendedMachine.MachineType, suggestedSize, recommendedWriteThroughputLimit)
		usage += fmt.Sprintf("- Disk Type %s with Machine Type %s with size %d has %.2f Mb Read Throughput estimation. Usage over the course of last week is min=%.2f%%, avg=%.2f%%, max=%.2f%%, so you only need %.2f Mb Read Throughput estimation. Disk Type %s with Machine Type %s with size %d has %.2f Mb Read Throughput estimation.\n", disk.DiskType, currentMachine.MachineType, PWrapperInt64(disk.DiskSize), readThroughputLimit, PWrapperDouble(readThroughputUsageMb.Min), PWrapperDouble(readThroughputUsageMb.Avg), PWrapperDouble(readThroughputUsageMb.Max), neededReadThroughput, suggestedType, recommendedMachine.MachineType, suggestedSize, recommendedReadThroughputLimit)
	} else {
		usage += fmt.Sprintf("- Disk Type %s with Machine Type %s with size %d has %.2f Mb Write Throughput estimation. Usage is not available. You need to install CloudWatch Agent on your instance to get this data. Disk Type %s with Machine Type %s with size %d has %.2f Mb Write Throughput estimation.\n", disk.DiskType, currentMachine.MachineType, PWrapperInt64(disk.DiskSize), writeThroughputLimit, suggestedType, recommendedMachine.MachineType, suggestedSize, recommendedWriteThroughputLimit)
		usage += fmt.Sprintf("- Disk Type %s with Machine Type %s with size %d has %.2f Mb Read Throughput estimation. Usage is not available. You need to install CloudWatch Agent on your instance to get this data. Disk Type %s with Machine Type %s with size %d has %.2f Mb Read Throughput estimation.\n", disk.DiskType, currentMachine.MachineType, PWrapperInt64(disk.DiskSize), readThroughputLimit, suggestedType, recommendedMachine.MachineType, suggestedSize, recommendedReadThroughputLimit)
	}

	needs := ""
	for k, v := range preferences {
		if gcp_compute.PreferenceDiskKey[k] == "" {
			continue
		}
		if v == nil {
			vl := extractFromGCPComputeDisk(disk, k)
			needs += fmt.Sprintf("- You asked %s to be same as the current instance value which is %v\n", k, vl)
		} else {
			needs += fmt.Sprintf("- You asked %s to be %s\n", k, v.GetValue())
		}
	}

	prompt := fmt.Sprintf(`
I'm giving recommendation on GCP Compute Disk right sizing. Based on user's usage and needs I have concluded that the best option for him is to use %s with size %d instead of %s with size %d. I need help summarizing the explanation into 280 characters (it's not a tweet! dont use hashtag!) while keeping these rules:
- mention the requirements from user side.
- for those fields which are changing make sure you mention the change.

Here's usage data:
%s

User's needs:
%s
`, suggestedType, suggestedSize, disk.DiskType, disk.DiskSize, usage, needs)

	resp, err := s.openaiSvc.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4TurboPreview,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("empty choices")
	}

	s.logger.Info("GPT results", zap.String("prompt", prompt), zap.String("result", strings.TrimSpace(resp.Choices[0].Message.Content)))

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func PWrapperDouble(v *wrapperspb.DoubleValue) float64 {
	if v == nil {
		return 0
	}
	return v.GetValue()
}

func PWrapperInt64(v *wrapperspb.Int64Value) int64 {
	if v == nil {
		return 0
	}
	return v.GetValue()
}

func PInt64Wrapper(v *int64) *wrapperspb.Int64Value {
	if v == nil {
		return nil
	}
	return &wrapperspb.Int64Value{Value: *v}
}

func pWrapperCalculateHeadroom(needed *wrapperspb.DoubleValue, percent int64) float64 {
	if needed == nil {
		return 0.0
	}
	v := needed.Value
	return v / (1.0 - (float64(percent) / 100.0))
}
