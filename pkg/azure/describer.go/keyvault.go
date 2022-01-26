package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	previewKeyvault "github.com/Azure/azure-sdk-for-go/services/preview/keyvault/mgmt/2020-04-01-preview/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func KeyVaultKey(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	keyVaultClient := keyvault.NewVaultsClient(subscription)
	keyVaultClient.Authorizer = authorizer
	maxResults := int32(100)

	client := keyvault.NewKeysClient(subscription)
	client.Authorizer = authorizer

	resultKV, err := keyVaultClient.List(ctx, &maxResults)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, vault := range resultKV.Values() {
			resourceGroup := strings.Split(*vault.ID, "/")[4]
			result, err := client.List(ctx, resourceGroup, *vault.Name)
			if err != nil {
				return nil, err
			}

			for {
				for _, v := range result.Values() {
					op, err := client.Get(ctx, resourceGroup, *vault.Name, *v.Name)
					if err != nil {
						return nil, err
					}

					// In some cases resource does not give any notFound error
					// instead of notFound error, it returns empty data
					if op.ID != nil {
						v = op
					}

					values = append(values, Resource{
						ID: *v.ID,
						Description: model.KeyVaultKeyDescription{
							Key: v,
							ResourceGroup: resourceGroup,
						},
					})
				}

				if !result.NotDone() {
					break
				}

				err = result.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}
			}
		}

		if !resultKV.NotDone() {
			break
		}

		err = resultKV.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func KeyVault(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	keyVaultClient := keyvault.NewVaultsClient(subscription)
	keyVaultClient.Authorizer = authorizer

	maxResults := int32(100)
	result, err := keyVaultClient.List(ctx, &maxResults)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for {
		for _, vault := range result.Values() {
			name := *vault.Name
			resourceGroup := strings.Split(*vault.ID, "/")[4]

			keyVaultGetOp, err := keyVaultClient.Get(ctx, resourceGroup, name)
			if err != nil {
				return nil, err
			}

			insightsListOp, err := insightsClient.List(ctx, *vault.ID)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID: *vault.ID,
				Description: model.KeyVaultDescription{
					Resource:                    vault,
					Vault:                       keyVaultGetOp,
					DiagnosticSettingsResources: insightsListOp.Value,
					ResourceGroup:               resourceGroup,
				},
			})
		}
		if !result.NotDone() {
			break
		}
		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}

func KeyVaultManagedHardwareSecurityModule(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := insights.NewDiagnosticSettingsClient(subscription)
	client.Authorizer = authorizer

	hsmClient := previewKeyvault.NewManagedHsmsClient(subscription)
	hsmClient.Authorizer = authorizer

	maxResults := int32(100)
	result, err := hsmClient.ListBySubscription(ctx, &maxResults)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, vault := range result.Values() {
			resourceGroup := strings.Split(*vault.ID, "/")[4]

			keyvaultListOp, err := client.List(ctx, *vault.ID)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID: *vault.ID,
				Description: model.KeyVaultManagedHardwareSecurityModuleDescription{
					ManagedHsm:                  vault,
					DiagnosticSettingsResources: keyvaultListOp.Value,
					ResourceGroup:               resourceGroup,
				},
			})
		}
		if !result.NotDone() {
			break
		}
		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}
