package onboard

import "github.com/opengovern/opengovernance/services/integration/model"

var defaultConnectionGroups = []model.ConnectionGroup{
	{
		Name:  "healthy",
		Query: `SELECT kaytu_id FROM kaytu_connections WHERE health_state = 'healthy'`,
	},
	{
		Name:  "unhealthy",
		Query: `SELECT kaytu_id FROM kaytu_connections WHERE health_state = 'unhealthy'`,
	},
}
