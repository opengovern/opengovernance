package enums

type DescribeTriggerType string

const (
	DescribeTriggerTypeInitialDiscovery DescribeTriggerType = "initial_discovery"
	DescribeTriggerTypeScheduled        DescribeTriggerType = "scheduled" // default
	DescribeTriggerTypeManual           DescribeTriggerType = "manual"
	DescribeTriggerTypeStack            DescribeTriggerType = "stack"
)
