package onboard

import "github.com/opengovern/opengovernance/services/integration/model"

var defaultConnectionGroups = []model.ConnectionGroup{
	{
		Name:  "healthy",
		Query: `SELECT og_id FROM og_connections WHERE health_state = 'healthy'`,
	},
	{
		Name:  "unhealthy",
		Query: `SELECT og_id FROM og_connections WHERE health_state = 'unhealthy'`,
	},
}
