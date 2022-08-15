package workspace

import "gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"

type Tier string

const (
	Tier_Free       Tier = "FREE"
	Tier_Teams      Tier = "TEAMS"
	Tier_Enterprise Tier = "ENTERPRISE"
)

func GetLimitsByTier(tier Tier) api.WorkspaceLimits {
	switch tier {
	case Tier_Free:
		return api.WorkspaceLimits{
			MaxUsers:       25,
			MaxConnections: 25,
			MaxResources:   2500,
		}
	case Tier_Teams:
		return api.WorkspaceLimits{
			MaxUsers:       250,
			MaxConnections: 2500,
			MaxResources:   1000000,
		}
	case Tier_Enterprise:
		return api.WorkspaceLimits{
			MaxUsers:       250,
			MaxConnections: 2500,
			MaxResources:   1000000,
		}
	}
	return api.WorkspaceLimits{}
}
