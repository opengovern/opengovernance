package describer

import (
	"context"
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/concurrency"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	previewKeyvault "github.com/Azure/azure-sdk-for-go/services/preview/keyvault/mgmt/2020-04-01-preview/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2021-04-01-preview/insights"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func KeyVaultKey(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	keyVaultClient := keyvault.NewVaultsClient(subscription)
	keyVaultClient.Authorizer = authorizer
	maxResults := int32(100)

	client := keyvault.NewKeysClient(subscription)
	client.Authorizer = authorizer

	resultKV, err := keyVaultClient.List(ctx, &maxResults)
	if err != nil {
		return nil, err
	}

	wpe := concurrency.NewWorkPool(4)
	var values []Resource
	for {
		for _, vaultLoop := range resultKV.Values() {
			vault := vaultLoop
			wpe.AddJob(func() (interface{}, error) {
				resourceGroup := strings.Split(*vault.ID, "/")[4]
				result, err := client.List(ctx, resourceGroup, *vault.Name)
				if err != nil {
					return nil, err
				}

				wp := concurrency.NewWorkPool(8)
				for {
					for _, v := range result.Values() {
						resourceGroupCopy := resourceGroup
						vaultCopy := vault
						vCopy := v
						wp.AddJob(func() (interface{}, error) {
							op, err := client.Get(ctx, resourceGroupCopy, *vaultCopy.Name, *vCopy.Name)
							if err != nil {
								return nil, err
							}

							// In some cases resource does not give any notFound error
							// instead of notFound error, it returns empty data
							if op.ID == nil {
								return nil, nil
							}

							return Resource{
								ID:       *vCopy.ID,
								Name:     *vCopy.Name,
								Location: *vCopy.Location,
								Description: model.KeyVaultKeyDescription{
									Vault:         vaultCopy,
									Key:           vCopy,
									ResourceGroup: resourceGroupCopy,
								},
							}, nil
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

				results := wp.Run()
				var vvv []Resource
				for _, r := range results {
					if r.Error != nil {
						return nil, err
					}
					if r.Value == nil {
						continue
					}
					vvv = append(vvv, r.Value.(Resource))
				}
				return vvv, nil
			})
		}

		if !resultKV.NotDone() {
			break
		}

		err = resultKV.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	results := wpe.Run()
	for _, result := range results {
		if result.Error != nil {
			return nil, err
		}
		if result.Value == nil {
			continue
		}
		values = append(values, result.Value.([]Resource)...)
	}

	if stream != nil {
		for _, resource := range values {
			if err := (*stream)(resource); err != nil {
				return nil, err
			}
		}
		values = nil
	}
	return values, nil
}

func KeyVault(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
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

			resource := Resource{
				ID:       *vault.ID,
				Name:     *vault.Name,
				Location: *vault.Location,
				Description: model.KeyVaultDescription{
					Resource:                    vault,
					Vault:                       keyVaultGetOp,
					DiagnosticSettingsResources: insightsListOp.Value,
					ResourceGroup:               resourceGroup,
				},
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
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

func DeletedVault(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	keyVaultClient := keyvault.NewVaultsClient(subscription)
	keyVaultClient.Authorizer = authorizer

	result, err := keyVaultClient.ListDeleted(ctx)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for {
		for _, vault := range result.Values() {
			resourceGroup := strings.Split(*vault.ID, "/")[4]

			resource := Resource{
				ID:       *vault.ID,
				Name:     *vault.Name,
				Location: *vault.Properties.Location,
				Description: model.KeyVaultDeletedVaultDescription{
					Vault:         vault,
					ResourceGroup: resourceGroup,
				},
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
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

func KeyVaultManagedHardwareSecurityModule(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
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

			resource := Resource{
				ID:       *vault.ID,
				Name:     *vault.Name,
				Location: *vault.Location,
				Description: model.KeyVaultManagedHardwareSecurityModuleDescription{
					ManagedHsm:                  vault,
					DiagnosticSettingsResources: keyvaultListOp.Value,
					ResourceGroup:               resourceGroup,
				},
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
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
