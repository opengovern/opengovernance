package recommendation

import (
	"fmt"
	"github.com/golang/protobuf/ptypes/wrappers"
	pb "github.com/kaytu-io/plugin-kubernetes/plugin/proto/src/golang"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"net/http"
	"sort"
	"strconv"
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

		recommended := pb.RightsizingKubernetesContainer{
			Name: container.Name,

			MemoryRequest: memoryTrimmedMean,
			MemoryLimit:   memoryMax,

			CpuRequest: cpuTrimmedMean,
			CpuLimit:   cpuMax,
		}

		if v, ok := preferences["CPUBreathingRoom"]; ok && v != nil {
			vPercent, err := strconv.ParseInt(v.Value, 10, 64)
			if err != nil {
				s.logger.Error("invalid CPUBreathingRoom value", zap.String("value", v.Value))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid CpuBreathingRoom value: %s", *v))
			}
			recommended.CpuLimit = calculateHeadroom(recommended.CpuLimit, vPercent)
			if recommended.CpuLimit < 0.1 {
				recommended.CpuLimit = 0.1
			}
			recommended.CpuRequest = calculateHeadroom(recommended.CpuRequest, vPercent)
			if recommended.CpuRequest < 0.1 {
				recommended.CpuRequest = 0.1
			}
		}

		if v, ok := preferences["MemoryBreathingRoom"]; ok && v != nil {
			vPercent, err := strconv.ParseInt(v.Value, 10, 64)
			if err != nil {
				s.logger.Error("invalid MemoryBreathingRoom value", zap.String("value", v.Value))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid MemoryBreathingRoom value: %s", *v))
			}
			recommended.MemoryLimit = calculateHeadroom(recommended.MemoryLimit, vPercent)
			if recommended.MemoryLimit == 0 {
				recommended.MemoryLimit = 100 * (1024 * 1024)
			}
			recommended.MemoryRequest = calculateHeadroom(recommended.MemoryRequest, vPercent)
			if recommended.MemoryRequest == 0 {
				recommended.MemoryRequest = 100 * (1024 * 1024)
			}
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
