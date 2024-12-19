package configs

import "github.com/opengovern/opencomply/services/integration/integration-type/interfaces"

var ResourceTypeConfigs = map[string]*interfaces.ResourceTypeConfiguration{
	"Github/Container/Package": {
		Name:            "Github/Container/Package",
		IntegrationType: IntegrationTypeGithubAccount,
		Description:     "",
		Params:          []interfaces.Param{},
	},
	"Github/Repository": {
		Name:            "Github/Repository",
		IntegrationType: IntegrationTypeGithubAccount,
		Description:     "",
		Params: []interfaces.Param{
			{
				Name:        "repository_name",
				Description: `Please provide the repo name (i.e. "internal-tools")`,
				Required:    false,
			},
		},
	},
}
