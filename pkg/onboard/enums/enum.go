package enums

type AssetDiscoveryMethodType string

const (
	AssetDiscoveryMethodTypeScheduled AssetDiscoveryMethodType = "scheduled"
)

type SourceHealthState string

const (
	SourceHealthStateHealthy          SourceHealthState = "healthy"
	SourceHealthStateUnhealthy        SourceHealthState = "unhealthy"
	SourceHealthStateInitialDiscovery SourceHealthState = "initial_discovery"
)

type SourceCreationMethod string

const (
	SourceCreationMethodManual SourceCreationMethod = "manual"
)
