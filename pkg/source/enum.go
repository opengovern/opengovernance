package source

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

type ConnectorDirectionType string

const (
	ConnectorDirectionTypeIngress ConnectorDirectionType = "ingress"
	ConnectorDirectionTypeEgress  ConnectorDirectionType = "egress"
	ConnectorDirectionTypeBoth    ConnectorDirectionType = "both"
)

type ConnectorStatus string

const (
	ConnectorStatusEnabled    ConnectorStatus = "enabled"
	ConnectorStatusDisabled   ConnectorStatus = "disabled"
	ConnectorStatusComingSoon ConnectorStatus = "coming_soon"
)
