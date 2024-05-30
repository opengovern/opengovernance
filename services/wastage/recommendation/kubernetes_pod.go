package recommendation

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"sort"
	"strconv"
)

func (s *Service) KubernetesPodRecommendation(
	pod corev1.Pod,
	metrics map[string]entity.KubernetesContainerMetrics,
	preferences map[string]*string,
) (*entity.KubernetesPodRightsizingRecommendation, error) {
	var containersRightsizing []entity.KubernetesContainerRightsizingRecommendation

	for _, container := range pod.Spec.Containers {
		current := entity.RightsizingKubernetesContainer{
			Name: container.Name,

			MemoryRequest: float64(container.Resources.Requests.Memory().Value()),
			MemoryLimit:   float64(container.Resources.Limits.Memory().Value()),

			CPURequest: float64(container.Resources.Requests.Cpu().MilliValue()),
			CPULimit:   float64(container.Resources.Limits.Cpu().MilliValue()),
		}

		if _, ok := metrics[container.Name]; !ok {
			containersRightsizing = append(containersRightsizing, entity.KubernetesContainerRightsizingRecommendation{
				Name: container.Name,

				Current: current,

				Description: "",
			})
			continue
		}

		cpuMax := getMetricMax(metrics[container.Name].CPU)
		cpuTrimmedMean, err := getTrimmedMean(metrics[container.Name].CPU, 0.1)
		if err != nil {
			return nil, err
		}
		memoryMax := getMetricMax(metrics[container.Name].Memory)
		memoryTrimmedMean, err := getTrimmedMean(metrics[container.Name].Memory, 0.1)
		if err != nil {
			return nil, err
		}

		recommended := entity.RightsizingKubernetesContainer{
			Name: container.Name,

			MemoryRequest: memoryTrimmedMean,

			CPURequest: cpuTrimmedMean,
		}
		if memoryMax != nil {
			recommended.MemoryLimit = *memoryMax
		}
		if cpuMax != nil {
			recommended.CPULimit = *cpuMax
		}

		if v, ok := preferences["CpuBreathingRoom"]; ok && v != nil {
			vPercent, err := strconv.ParseInt(*v, 10, 64)
			if err != nil {
				s.logger.Error("invalid CpuBreathingRoom value", zap.String("value", *v))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid CpuBreathingRoom value: %s", *v))
			}
			recommended.CPULimit = calculateHeadroom(recommended.CPULimit, vPercent)
		}

		if v, ok := preferences["MemoryBreathingRoom"]; ok && v != nil {
			vPercent, err := strconv.ParseInt(*v, 10, 64)
			if err != nil {
				s.logger.Error("invalid MemoryBreathingRoom value", zap.String("value", *v))
				return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid MemoryBreathingRoom value: %s", *v))
			}
			recommended.MemoryLimit = calculateHeadroom(recommended.MemoryLimit, vPercent)
		}

		containersRightsizing = append(containersRightsizing, entity.KubernetesContainerRightsizingRecommendation{
			Name: container.Name,

			Current:     current,
			Recommended: &recommended,

			MemoryTrimmedMean: &memoryTrimmedMean,
			MemoryMax:         memoryMax,
			CPUTrimmedMean:    &cpuTrimmedMean,
			CPUMax:            cpuMax,

			Description: "",
		})
	}

	return &entity.KubernetesPodRightsizingRecommendation{
		Name: pod.Name,

		ContainersRightsizing: containersRightsizing,
	}, nil
}

func getMetricMax(data map[string]float64) *float64 {
	var dMax *float64
	for _, v := range data {
		if dMax == nil || v > *dMax {
			dMax = &v
		}
	}
	return dMax
}

func getTrimmedMean(data map[string]float64, trimPercentage float64) (float64, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("empty map provided")
	}

	values := make([]float64, 0, len(data))
	for _, v := range data {
		values = append(values, v)
	}

	sort.Float64s(values)

	numToTrim := int(trimPercentage * float64(len(data)))
	trimmedValues := values[numToTrim : len(values)-numToTrim]

	var sum float64
	for _, v := range trimmedValues {
		sum += v
	}
	return sum / float64(len(trimmedValues)), nil
}
