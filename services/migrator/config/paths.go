package config

const (
	ControlEnrichmentGitPath = "/tmp/loader-control-enrichment-git"

	AnalyticsGitPath          = "/tmp/loader-analytics-git"
	ComplianceGitPath         = AnalyticsGitPath + "/compliance"
	QueriesGitPath            = AnalyticsGitPath + "/cloud-infrastructure-queries"
	InsightsGitPath           = AnalyticsGitPath + "/insights"
	ResourceCollectionGitPath = AnalyticsGitPath + "/resource-collections"
	ConnectionGroupGitPath    = AnalyticsGitPath + "/connection_groups"
)

const (
	InsightsSubPath      = "insights"
	InsightGroupsSubPath = "insight_groups"
)
