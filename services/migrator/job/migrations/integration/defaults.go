package integration

import (
	"github.com/opengovern/opengovernance/services/integration/models"
)

var defaultIntegrationGroups = []models.IntegrationGroup{
	{
		Name:  "active",
		Query: `SELECT integration_id FROM og_integrations WHERE state = 'active'`,
	},
	{
		Name:  "inactive",
		Query: `SELECT integration_id FROM og_integrations WHERE state = 'inactive'`,
	},
	{
		Name:  "archived",
		Query: `SELECT integration_id FROM og_integrations WHERE state = 'archived'`,
	},
}
