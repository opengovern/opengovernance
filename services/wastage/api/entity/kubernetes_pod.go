package entity

import (
	corev1 "k8s.io/api/core/v1"
)

type RightsizingKubernetesContainer struct {
	Name string `json:"name"`

	MemoryRequest float64 `json:"memoryRequest"`
	MemoryLimit   float64 `json:"memoryLimit"`

	CPURequest float64 `json:"cpuRequest"`
	CPULimit   float64 `json:"cpuLimit"`
}

type KubernetesContainerRightsizingRecommendation struct {
	Name string `json:"name"`

	Current     RightsizingKubernetesContainer  `json:"current"`
	Recommended *RightsizingKubernetesContainer `json:"recommended"`

	MemoryTrimmedMean *float64 `json:"memoryTrimmedMean"`
	MemoryMax         *float64 `json:"memoryMax"`
	CPUTrimmedMean    *float64 `json:"cpuTrimmedMean"`
	CPUMax            *float64 `json:"cpuMax"`

	Description string `json:"description"`
}

type KubernetesPodRightsizingRecommendation struct {
	Name string `json:"name"`

	ContainersRightsizing []KubernetesContainerRightsizingRecommendation `json:"containersRightsizing"`
}

type KubernetesContainerMetrics struct {
	CPU    map[string]float64 `json:"cpu"`
	Memory map[string]float64 `json:"memory"`
}

type KubernetesPodWastageRequest struct {
	RequestId      *string                               `json:"requestId"`
	CliVersion     *string                               `json:"cliVersion"`
	Identification map[string]string                     `json:"identification"`
	Pod            corev1.Pod                            `json:"pod"`
	Namespace      string                                `json:"namespace"`
	Preferences    map[string]*string                    `json:"preferences"`
	Metrics        map[string]KubernetesContainerMetrics `json:"metrics"` // container name -> metrics
	Loading        bool                                  `json:"loading"`
}

type KubernetesPodWastageResponse struct {
	RightSizing KubernetesPodRightsizingRecommendation `json:"rightSizing"`
}
