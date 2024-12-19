package configs

import "github.com/opengovern/opencomply/services/integration/integration-type/interfaces"

var ResourceTypeConfigs = map[string]*interfaces.ResourceTypeConfiguration{
	"Github/Container/Package": &interfaces.ResourceTypeConfiguration{
		Name:            "Github/Container/Package",
		IntegrationType: IntegrationTypeGithubAccount,
		Description:     "",
		Params:          []interfaces.Param{},
	},
}
