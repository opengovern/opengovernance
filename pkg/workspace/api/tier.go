package api

type Tier string

const (
	Tier_Free       Tier = "FREE"
	Tier_Teams      Tier = "TEAMS"
	Tier_Enterprise Tier = "ENTERPRISE"
)

func GetLimitsByTier(tier Tier) WorkspaceLimits {
	switch tier {
	case Tier_Free:
		return WorkspaceLimits{
			MaxUsers:       25,
			MaxConnections: 25,
			MaxResources:   2500,
		}
	case Tier_Teams:
		return WorkspaceLimits{
			MaxUsers:       250,
			MaxConnections: 2500,
			MaxResources:   1000000,
		}
	case Tier_Enterprise:
		return WorkspaceLimits{
			MaxUsers:       250,
			MaxConnections: 25000,
			MaxResources:   100000000,
		}
	}
	return WorkspaceLimits{}
}
