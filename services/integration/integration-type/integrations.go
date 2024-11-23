package integration_type

import (
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/opengovernance/services/integration/integration-type/aws-account"
	awsConfigs "github.com/opengovern/opengovernance/services/integration/integration-type/aws-account/configs"
	"github.com/opengovern/opengovernance/services/integration/integration-type/azure-subscription"
	azureConfigs "github.com/opengovern/opengovernance/services/integration/integration-type/azure-subscription/configs"
	cloudflare_account "github.com/opengovern/opengovernance/services/integration/integration-type/cloudflare-account"
	cloudflareConfigs "github.com/opengovern/opengovernance/services/integration/integration-type/cloudflare-account/configs"
	"github.com/opengovern/opengovernance/services/integration/integration-type/digitalocean-team"
	digitalOceanConfigs "github.com/opengovern/opengovernance/services/integration/integration-type/digitalocean-team/configs"
	"github.com/opengovern/opengovernance/services/integration/integration-type/entra-id-directory"
	entraidConfigs "github.com/opengovern/opengovernance/services/integration/integration-type/entra-id-directory/configs"
	github_account "github.com/opengovern/opengovernance/services/integration/integration-type/github-account"
	githubConfigs "github.com/opengovern/opengovernance/services/integration/integration-type/github-account/configs"
	"github.com/opengovern/opengovernance/services/integration/integration-type/interfaces"
	"strings"
)

const (
	IntegrationTypeAWSAccount        = awsConfigs.IntegrationTypeAwsCloudAccount
	IntegrationTypeAzureSubscription = azureConfigs.IntegrationTypeAzureSubscription
	IntegrationTypeEntraIdDirectory  = entraidConfigs.IntegrationTypeEntraidDirectory
	IntegrationTypeGithubAccount     = githubConfigs.IntegrationTypeGithubAccount
	IntegrationTypeDigitalOceanTeam  = digitalOceanConfigs.IntegrationTypeDigitalOceanTeam
	IntegrationTypeCloudflareAccount = cloudflareConfigs.IntegrationNameCloudflareAccount
)

var AllIntegrationTypes = []integration.Type{
	IntegrationTypeAWSAccount,
	IntegrationTypeAzureSubscription,
	IntegrationTypeEntraIdDirectory,
	IntegrationTypeGithubAccount,
	IntegrationTypeDigitalOceanTeam,
	IntegrationTypeCloudflareAccount,
}

var IntegrationTypes = map[integration.Type]interfaces.IntegrationType{
	IntegrationTypeAWSAccount:        &aws_account.AwsCloudAccountIntegration{},
	IntegrationTypeAzureSubscription: &azure_subscription.AzureSubscriptionIntegration{},
	IntegrationTypeEntraIdDirectory:  &entra_id_directory.EntraIdDirectoryIntegration{},
	IntegrationTypeGithubAccount:     &github_account.GithubAccountIntegration{},
	IntegrationTypeDigitalOceanTeam:  &digitalocean_team.DigitaloceanTeamIntegration{},
	IntegrationTypeCloudflareAccount: &cloudflare_account.CloudFlareAccountIntegration{},
}

func ParseType(str string) integration.Type {
	str = strings.ToLower(str)
	for _, t := range AllIntegrationTypes {
		if str == t.String() {
			return t
		}
	}
	return ""
}

func ParseTypes(str []string) []integration.Type {
	result := make([]integration.Type, 0, len(str))
	for _, s := range str {
		t := ParseType(s)
		if t == "" {
			continue
		}
		result = append(result, t)
	}
	return result
}

func UnparseTypes(types []integration.Type) []string {
	result := make([]string, 0, len(types))
	for _, t := range types {
		result = append(result, t.String())
	}
	return result
}
