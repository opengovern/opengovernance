package recommendation

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	pb "github.com/kaytu-io/plugin-kubernetes-internal/plugin/proto/src/golang"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func (s *Service) KubernetesPodRecommendation(
	pod pb.KubernetesPod,
	metrics map[string]*pb.KubernetesContainerMetrics,
	preferences map[string]*wrappers.StringValue,
) (*pb.KubernetesPodRightsizingRecommendation, error) {
	var containersRightsizing []*pb.KubernetesContainerRightsizingRecommendation

	for _, container := range pod.Containers {
		current := pb.RightsizingKubernetesContainer{
			Name: container.Name,

			MemoryRequest: container.MemoryRequest,
			MemoryLimit:   container.MemoryLimit,

			CpuRequest: container.CpuRequest,
			CpuLimit:   container.CpuLimit,
		}

		if _, ok := metrics[container.Name]; !ok {
			containersRightsizing = append(containersRightsizing, &pb.KubernetesContainerRightsizingRecommendation{
				Name: container.Name,

				Current: &current,

				Description: "",
			})
			continue
		}

		cpuMax := getMetricMax(metrics[container.Name].Cpu)
		cpuTrimmedMean := getTrimmedMean(metrics[container.Name].Cpu, 0.1)
		memoryMax := getMetricMax(metrics[container.Name].Memory)
		memoryTrimmedMean := getTrimmedMean(metrics[container.Name].Memory, 0.1)

		if pod.Name == "contour-envoy-kl545" {
			s.logger.Info("contour-envoy-kl545 usage1", zap.Any("cpuMax", cpuMax), zap.String("container", container.Name),
				zap.Any("cpuTrimmedMean", cpuTrimmedMean), zap.Any("memoryMax", memoryMax), zap.Any("memoryTrimmedMean", memoryTrimmedMean))
		}

		recommended := pb.RightsizingKubernetesContainer{
			Name: container.Name,

			MemoryRequest: memoryTrimmedMean,
			MemoryLimit:   memoryMax,

			CpuRequest: cpuTrimmedMean,
			CpuLimit:   cpuMax,
		}

		if pod.Name == "contour-envoy-kl545" {
			s.logger.Info("contour-envoy-kl545 recommended1", zap.String("container", container.Name),
				zap.Any("CpuLimit", recommended.CpuLimit), zap.Any("CpuRequest", recommended.CpuRequest), zap.Any("MemoryLimit", recommended.MemoryLimit), zap.Any("MemoryRequest", recommended.MemoryRequest))
		}

		if v, ok := preferences["CPURequestBreathingRoom"]; ok && v != nil {
			vPercent, err := strconv.ParseInt(v.Value, 10, 64)
			if err != nil {
				s.logger.Error("invalid CPURequestBreathingRoom value", zap.String("value", v.Value))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid CPURequestBreathingRoom value: %s", v.Value))
			}
			recommended.CpuRequest = calculateHeadroom(recommended.CpuRequest, vPercent)
			if recommended.CpuRequest < 0.1 {
				recommended.CpuRequest = 0.1
			}
		}
		if v, ok := preferences["CPULimitBreathingRoom"]; ok && v != nil {
			vPercent, err := strconv.ParseInt(v.Value, 10, 64)
			if err != nil {
				s.logger.Error("invalid CPULimitBreathingRoom value", zap.String("value", v.Value))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid CpuLimitBreathingRoom value: %s", v.Value))
			}
			recommended.CpuLimit = calculateHeadroom(recommended.CpuLimit, vPercent)
			if recommended.CpuLimit < 0.1 {
				recommended.CpuLimit = 0.1
			}
		}

		if v, ok := preferences["MemoryRequestBreathingRoom"]; ok && v != nil {
			vPercent, err := strconv.ParseInt(v.Value, 10, 64)
			if err != nil {
				s.logger.Error("invalid MemoryRequestBreathingRoom value", zap.String("value", v.Value))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid MemoryRequestBreathingRoom value: %s", v.Value))
			}
			recommended.MemoryRequest = calculateHeadroom(recommended.MemoryRequest, vPercent)
			if recommended.MemoryRequest == 0 {
				recommended.MemoryRequest = 100 * (1024 * 1024)
			}
		}
		if v, ok := preferences["MemoryLimitBreathingRoom"]; ok && v != nil {
			vPercent, err := strconv.ParseInt(v.Value, 10, 64)
			if err != nil {
				s.logger.Error("invalid MemoryLimitBreathingRoom value", zap.String("value", v.Value))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid MemoryLimitBreathingRoom value: %s", v.Value))
			}
			recommended.MemoryLimit = calculateHeadroom(recommended.MemoryLimit, vPercent)
			if recommended.MemoryLimit == 0 {
				recommended.MemoryLimit = 100 * (1024 * 1024)
			}
		}
		if v, ok := preferences["MinCpuRequest"]; ok && v != nil {
			minCpuRequest, err := strconv.ParseFloat(v.Value, 64)
			if err != nil {
				s.logger.Error("invalid MinCpuRequest value", zap.String("value", v.Value))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid MinCpuRequest value: %s", v.Value))
			}
			if recommended.CpuRequest < minCpuRequest {
				recommended.CpuRequest = minCpuRequest
			}
		}
		if v, ok := preferences["MinMemoryRequest"]; ok && v != nil {
			minMemoryRequest, err := strconv.ParseFloat(v.Value, 64)
			if err != nil {
				s.logger.Error("invalid MinMemoryRequest value", zap.String("value", v.Value))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid MinMemoryRequest value: %s", v.Value))
			}
			if recommended.MemoryRequest < minMemoryRequest {
				recommended.MemoryRequest = minMemoryRequest * (1024 * 1024)
			}
		}
		if v, ok := preferences["LeaveCPULimitEmpty"]; ok && v != nil {
			leaveCPULimitEmpty, err := strconv.ParseBool(v.Value)
			if err != nil {
				s.logger.Error("invalid LeaveCPULimitEmpty value", zap.String("value", v.Value))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid LeaveCPULimitEmpty value: %s", v.Value))
			}

			if leaveCPULimitEmpty {
				recommended.CpuRequest = recommended.CpuLimit
				recommended.CpuLimit = current.CpuLimit
			}
		}
		if v, ok := preferences["EqualMemoryRequestLimit"]; ok && v != nil {
			equalMemoryRequestLimit, err := strconv.ParseBool(v.Value)
			if err != nil {
				s.logger.Error("invalid EqualMemoryRequestLimit value", zap.String("value", v.Value))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid EqualMemoryRequestLimit value: %s", v.Value))
			}
			if equalMemoryRequestLimit {
				recommended.MemoryRequest = recommended.MemoryLimit
			}
		}

		if pod.Name == "contour-envoy-kl545" {
			s.logger.Info("contour-envoy-kl545 recommended2", zap.String("container", container.Name),
				zap.Any("CpuLimit", recommended.CpuLimit), zap.Any("CpuRequest", recommended.CpuRequest), zap.Any("MemoryLimit", recommended.MemoryLimit), zap.Any("MemoryRequest", recommended.MemoryRequest))
		}

		var usageMemoryTrimmedMean, usageMemoryMax, usageCpuTrimmedMean, usageCpuMax *wrappers.DoubleValue
		if len(metrics[container.Name].Cpu) > 0 {
			usageCpuTrimmedMean = wrapperspb.Double(cpuTrimmedMean)
			usageCpuMax = wrapperspb.Double(cpuMax)
		}
		if len(metrics[container.Name].Memory) > 0 {
			usageMemoryTrimmedMean = wrapperspb.Double(memoryTrimmedMean)
			usageMemoryMax = wrapperspb.Double(memoryMax)
		}

		containersRightsizing = append(containersRightsizing, &pb.KubernetesContainerRightsizingRecommendation{
			Name: container.Name,

			Current:     &current,
			Recommended: &recommended,

			MemoryTrimmedMean: usageMemoryTrimmedMean,
			MemoryMax:         usageMemoryMax,
			CpuTrimmedMean:    usageCpuTrimmedMean,
			CpuMax:            usageCpuMax,

			Description: "",
		})
	}

	return &pb.KubernetesPodRightsizingRecommendation{
		Name: pod.Name,

		ContainerResizing: containersRightsizing,
	}, nil
}

func (s *Service) KubernetesDeploymentRecommendation(
	deployment pb.KubernetesDeployment,
	metrics map[string]*pb.KubernetesPodMetrics,
	preferences map[string]*wrappers.StringValue,
) (*pb.KubernetesDeploymentRightsizingRecommendation, error) {
	result := pb.KubernetesDeploymentRightsizingRecommendation{
		Name:                 deployment.Name,
		ContainerResizing:    nil,
		PodContainerResizing: make(map[string]*pb.KubernetesPodRightsizingRecommendation),
	}

	overallMetrics := make(map[string]*pb.KubernetesContainerMetrics)
	for podName, podMetrics := range metrics {
		for containerName, containerMetrics := range podMetrics.Metrics {
			containerMetrics := containerMetrics
			overallMetrics[containerName] = mergeContainerMetrics(overallMetrics[containerName], containerMetrics, func(aa, bb float64) float64 {
				return max(aa, bb)
			})
		}

		podContainerResizing, err := s.KubernetesPodRecommendation(pb.KubernetesPod{
			Id:         podName,
			Name:       podName,
			Containers: deployment.Containers,
		}, podMetrics.Metrics, preferences)
		if err != nil {
			s.logger.Error("failed to get kubernetes pod recommendation", zap.Error(err))
			return nil, err
		}
		result.PodContainerResizing[podName] = podContainerResizing
	}

	containerResizings, err := s.KubernetesPodRecommendation(pb.KubernetesPod{
		Id:         deployment.Name,
		Name:       deployment.Name,
		Containers: deployment.Containers,
	}, overallMetrics, preferences)
	if err != nil {
		s.logger.Error("failed to get kubernetes pod recommendation", zap.Error(err))
		return nil, err
	}
	result.ContainerResizing = containerResizings.ContainerResizing
	for _, containerResizing := range result.ContainerResizing {
		containerResizing := containerResizing
		for podName, podContainerResizings := range result.PodContainerResizing {
			podContainerResizings := podContainerResizings
			for i, podContainerResizing := range podContainerResizings.ContainerResizing {
				podContainerResizing := podContainerResizing
				if podContainerResizing == nil || podContainerResizing.Name != containerResizing.Name {
					continue
				}
				podContainerResizing.Current = containerResizing.Current
				podContainerResizing.Recommended = containerResizing.Recommended
				podContainerResizing.Description = containerResizing.Description
				podContainerResizings.ContainerResizing[i] = podContainerResizing
			}
			result.PodContainerResizing[podName] = podContainerResizings
		}
	}

	return &result, nil
}

func (s *Service) KubernetesStatefulsetRecommendation(
	statefulset pb.KubernetesStatefulset,
	metrics map[string]*pb.KubernetesPodMetrics,
	preferences map[string]*wrappers.StringValue,
) (*pb.KubernetesStatefulsetRightsizingRecommendation, error) {
	result := pb.KubernetesStatefulsetRightsizingRecommendation{
		Name:                 statefulset.Name,
		ContainerResizing:    nil,
		PodContainerResizing: make(map[string]*pb.KubernetesPodRightsizingRecommendation),
	}

	overallMetrics := make(map[string]*pb.KubernetesContainerMetrics)
	for podName, podMetrics := range metrics {
		for containerName, containerMetrics := range podMetrics.Metrics {
			containerMetrics := containerMetrics
			overallMetrics[containerName] = mergeContainerMetrics(overallMetrics[containerName], containerMetrics, func(aa, bb float64) float64 {
				return max(aa, bb)
			})
		}

		podContainerResizing, err := s.KubernetesPodRecommendation(pb.KubernetesPod{
			Id:         podName,
			Name:       podName,
			Containers: statefulset.Containers,
		}, podMetrics.Metrics, preferences)
		if err != nil {
			s.logger.Error("failed to get kubernetes pod recommendation", zap.Error(err))
			return nil, err
		}
		result.PodContainerResizing[podName] = podContainerResizing
	}

	containerResizings, err := s.KubernetesPodRecommendation(pb.KubernetesPod{
		Id:         statefulset.Name,
		Name:       statefulset.Name,
		Containers: statefulset.Containers,
	}, overallMetrics, preferences)
	if err != nil {
		s.logger.Error("failed to get kubernetes pod recommendation", zap.Error(err))
		return nil, err
	}
	result.ContainerResizing = containerResizings.ContainerResizing
	for _, containerResizing := range result.ContainerResizing {
		containerResizing := containerResizing
		for podName, podContainerResizings := range result.PodContainerResizing {
			podContainerResizings := podContainerResizings
			for i, podContainerResizing := range podContainerResizings.ContainerResizing {
				podContainerResizing := podContainerResizing
				if podContainerResizing == nil || podContainerResizing.Name != containerResizing.Name {
					continue
				}
				podContainerResizing.Current = containerResizing.Current
				podContainerResizing.Recommended = containerResizing.Recommended
				podContainerResizing.Description = containerResizing.Description
				podContainerResizings.ContainerResizing[i] = podContainerResizing
			}
			result.PodContainerResizing[podName] = podContainerResizings
		}
	}

	return &result, nil
}

func (s *Service) KubernetesDaemonsetRecommendation(
	daemonset pb.KubernetesDaemonset,
	metrics map[string]*pb.KubernetesPodMetrics,
	preferences map[string]*wrappers.StringValue,
) (*pb.KubernetesDaemonsetRightsizingRecommendation, error) {
	result := pb.KubernetesDaemonsetRightsizingRecommendation{
		Name:                 daemonset.Name,
		ContainerResizing:    nil,
		PodContainerResizing: make(map[string]*pb.KubernetesPodRightsizingRecommendation),
	}

	overallMetrics := make(map[string]*pb.KubernetesContainerMetrics)
	for podName, podMetrics := range metrics {
		for containerName, containerMetrics := range podMetrics.Metrics {
			containerMetrics := containerMetrics
			overallMetrics[containerName] = mergeContainerMetrics(overallMetrics[containerName], containerMetrics, func(aa, bb float64) float64 {
				return max(aa, bb)
			})
		}

		podContainerResizing, err := s.KubernetesPodRecommendation(pb.KubernetesPod{
			Id:         podName,
			Name:       podName,
			Containers: daemonset.Containers,
		}, podMetrics.Metrics, preferences)
		if err != nil {
			s.logger.Error("failed to get kubernetes pod recommendation", zap.Error(err))
			return nil, err
		}
		result.PodContainerResizing[podName] = podContainerResizing
	}

	containerResizings, err := s.KubernetesPodRecommendation(pb.KubernetesPod{
		Id:         daemonset.Name,
		Name:       daemonset.Name,
		Containers: daemonset.Containers,
	}, overallMetrics, preferences)
	if err != nil {
		s.logger.Error("failed to get kubernetes pod recommendation", zap.Error(err))
		return nil, err
	}
	result.ContainerResizing = containerResizings.ContainerResizing
	for _, containerResizing := range result.ContainerResizing {
		containerResizing := containerResizing
		for podName, podContainerResizings := range result.PodContainerResizing {
			podContainerResizings := podContainerResizings
			for i, podContainerResizing := range podContainerResizings.ContainerResizing {
				podContainerResizing := podContainerResizing
				if podContainerResizing == nil || podContainerResizing.Name != containerResizing.Name {
					continue
				}
				podContainerResizing.Current = containerResizing.Current
				podContainerResizing.Recommended = containerResizing.Recommended
				podContainerResizing.Description = containerResizing.Description
				podContainerResizings.ContainerResizing[i] = podContainerResizing
			}
			result.PodContainerResizing[podName] = podContainerResizings
		}
	}

	return &result, nil
}

func (s *Service) KubernetesJobRecommendation(
	job pb.KubernetesJob,
	metrics map[string]*pb.KubernetesPodMetrics,
	preferences map[string]*wrappers.StringValue,
) (*pb.KubernetesJobRightsizingRecommendation, error) {
	result := pb.KubernetesJobRightsizingRecommendation{
		Name:                 job.Name,
		ContainerResizing:    nil,
		PodContainerResizing: make(map[string]*pb.KubernetesPodRightsizingRecommendation),
	}

	overallMetrics := make(map[string]*pb.KubernetesContainerMetrics)
	for podName, podMetrics := range metrics {
		for containerName, containerMetrics := range podMetrics.Metrics {
			containerMetrics := containerMetrics
			overallMetrics[containerName] = mergeContainerMetrics(overallMetrics[containerName], containerMetrics, func(aa, bb float64) float64 {
				return max(aa, bb)
			})
		}

		podContainerResizing, err := s.KubernetesPodRecommendation(pb.KubernetesPod{
			Id:         podName,
			Name:       podName,
			Containers: job.Containers,
		}, podMetrics.Metrics, preferences)
		if err != nil {
			s.logger.Error("failed to get kubernetes pod recommendation", zap.Error(err))
			return nil, err
		}
		result.PodContainerResizing[podName] = podContainerResizing
	}

	containerResizings, err := s.KubernetesPodRecommendation(pb.KubernetesPod{
		Id:         job.Name,
		Name:       job.Name,
		Containers: job.Containers,
	}, overallMetrics, preferences)
	if err != nil {
		s.logger.Error("failed to get kubernetes pod recommendation", zap.Error(err))
		return nil, err
	}
	result.ContainerResizing = containerResizings.ContainerResizing
	for _, containerResizing := range result.ContainerResizing {
		containerResizing := containerResizing
		for podName, podContainerResizings := range result.PodContainerResizing {
			podContainerResizings := podContainerResizings
			for i, podContainerResizing := range podContainerResizings.ContainerResizing {
				podContainerResizing := podContainerResizing
				if podContainerResizing == nil || podContainerResizing.Name != containerResizing.Name {
					continue
				}
				podContainerResizing.Current = containerResizing.Current
				podContainerResizing.Recommended = containerResizing.Recommended
				podContainerResizing.Description = containerResizing.Description
				podContainerResizings.ContainerResizing[i] = podContainerResizing
			}
			result.PodContainerResizing[podName] = podContainerResizings
		}
	}

	return &result, nil
}

func (s *Service) calculateEksNodeCost(ctx context.Context, node pb.KubernetesNode) (float64, error) {
	var instanceType, instanceRegion, instanceAvailabilityZone, instanceOs string

	for _, v := range []string{"node.kubernetes.io/instance-type", "beta.kubernetes.io/instance-type"} {
		var ok bool
		instanceType, ok = node.Labels[v]
		if ok {
			break
		}
	}
	if instanceType == "" {
		return 0, status.Errorf(codes.InvalidArgument, "Cannot determine the instance type for the node")
	}

	for _, v := range []string{"topology.kubernetes.io/region", "failure-domain.beta.kubernetes.io/region"} {
		var ok bool
		instanceRegion, ok = node.Labels[v]
		if ok {
			break
		}
	}
	if instanceRegion == "" {
		return 0, status.Errorf(codes.InvalidArgument, "Cannot determine the region for the node")
	}

	for _, v := range []string{"topology.kubernetes.io/zone", "failure-domain.beta.kubernetes.io/zone"} {
		var ok bool
		instanceAvailabilityZone, ok = node.Labels[v]
		if ok {
			break
		}
	}
	if instanceAvailabilityZone == "" {
		return 0, status.Errorf(codes.InvalidArgument, "Cannot determine the availability zone for the node")
	}

	for _, v := range []string{"kubernetes.io/os", "beta.kubernetes.io/os"} {
		var ok bool
		instanceOs, ok = node.Labels[v]
		if ok {
			break
		}
	}
	if instanceOs == "" {
		return 0, status.Errorf(codes.InvalidArgument, "Cannot determine the operating system for the node")
	}

	capacityType, ok := node.Labels["eks.amazonaws.com/capacityType"]
	if !ok {
		capacityType = "ON_DEMAND" // or throw an error?
	}

	instance := entity.EC2Instance{
		HashedInstanceId:  node.Id,
		State:             types.InstanceStateNameRunning,
		InstanceType:      types.InstanceType(instanceType),
		Platform:          "",
		UsageOperation:    "",
		InstanceLifecycle: types.InstanceLifecycleTypeScheduled,
		Placement: &entity.EC2Placement{
			Tenancy:          "default",
			AvailabilityZone: instanceAvailabilityZone,
		},
	}
	if capacityType == "SPOT" {
		instance.InstanceLifecycle = types.InstanceLifecycleTypeSpot
	}
	switch instanceOs {
	case "linux":
		instance.Platform = "Linux/UNIX"
		instance.UsageOperation = "RunInstances"
	case "windows":
		instance.Platform = "Windows"
		instance.UsageOperation = "RunInstances:0002"
	default:
		return 0, status.Errorf(codes.InvalidArgument, "Unsupported operating system for the node: %s", instanceOs)
	}

	cost, _, err := s.costSvc.GetEC2InstanceCost(ctx, instanceRegion, instance, nil, nil)
	if err != nil {
		s.logger.Error("failed to get ec2 instance cost", zap.Error(err))
		return 0, err
	}

	return cost, nil
}

func (s *Service) KubernetesNodeCost(ctx context.Context, node pb.KubernetesNode) (float64, error) {
	for labelKey, _ := range node.Labels {
		labelKey := strings.ToLower(labelKey)
		switch {
		case strings.HasPrefix(labelKey, "eks.amazonaws.com/"):
			return s.calculateEksNodeCost(ctx, node)
		case strings.HasPrefix(labelKey, "kubernetes.azure.com/"):
			return 0, status.Errorf(codes.InvalidArgument, "AKS cluster node costs are not supported")
			// TODO @Arta GCP case
		}
	}
	return 0, status.Errorf(codes.InvalidArgument, "Cannot determine the cloud provider for the node")
}

func mergeContainerMetrics(a *pb.KubernetesContainerMetrics, b *pb.KubernetesContainerMetrics, mergeF func(aa, bb float64) float64) *pb.KubernetesContainerMetrics {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	result := &pb.KubernetesContainerMetrics{
		Cpu:    make(map[string]float64),
		Memory: make(map[string]float64),
	}

	for k, v := range a.Cpu {
		result.Cpu[k] = v
	}
	for k, v := range b.Cpu {
		if _, ok := result.Cpu[k]; ok {
			result.Cpu[k] = mergeF(result.Cpu[k], v)
		} else {
			result.Cpu[k] = v
		}
	}

	for k, v := range a.Memory {
		result.Memory[k] = v
	}
	for k, v := range b.Memory {
		if _, ok := result.Memory[k]; ok {
			result.Memory[k] = mergeF(result.Memory[k], v)
		} else {
			result.Memory[k] = v
		}
	}

	return result
}

func getMetricMax(data map[string]float64) float64 {
	if len(data) == 0 {
		return 0
	}
	dMax := float64(0)
	for _, v := range data {
		if v > dMax {
			dMax = v
		}
	}
	return dMax
}

func getTrimmedMean(data map[string]float64, trimPercentage float64) float64 {
	if len(data) == 0 {
		return 0
	}

	values := make([]float64, 0, len(data))
	for _, v := range data {
		values = append(values, v)
	}

	sort.Float64s(values)

	numToTrim := int(trimPercentage * float64(len(data)) / 2)
	trimmedValues := values[numToTrim : len(values)-numToTrim]

	var sum float64
	for _, v := range trimmedValues {
		sum += v
	}
	return float64(sum) / float64(len(trimmedValues))
}
