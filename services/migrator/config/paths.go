package config

const (
	ControlEnrichmentGitPath = "/tmp/loader-control-enrichment-git"

	ConfigzGitPath            = "/tmp/loader-analytics-git"
	AssetsGitPath             = ConfigzGitPath + "/analytics/cloud-infra"
	SpendGitPath              = ConfigzGitPath + "/analytics/cloud-spend"
	FinderPopularGitPath      = ConfigzGitPath + "/query-engine/popular-queries"
	FinderOthersGitPath       = ConfigzGitPath + "/query-engine/other-queries"
	ComplianceGitPath         = ConfigzGitPath + "/policies"
	ResourceCollectionGitPath = ConfigzGitPath + "/resource-collections"
	ConnectionGroupGitPath    = ConfigzGitPath + "/connection_groups"
)
