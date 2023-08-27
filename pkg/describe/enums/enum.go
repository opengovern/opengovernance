package enums

type DescribeTriggerType string

const (
	DescribeTriggerTypeInitialDiscovery  DescribeTriggerType = "initial_discovery"
	DescribeTriggerTypeCostFullDiscovery DescribeTriggerType = "cost_full_discovery" // default
	DescribeTriggerTypeScheduled         DescribeTriggerType = "scheduled"           // default
	DescribeTriggerTypeManual            DescribeTriggerType = "manual"
	DescribeTriggerTypeStack             DescribeTriggerType = "stack"
)
