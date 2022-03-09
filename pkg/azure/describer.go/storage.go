package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/queue/queues"
	"github.com/tombuildsstuff/giovanni/storage/2019-12-12/blob/accounts"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func StorageContainer(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := storage.NewBlobContainersClient(subscription)
	client.Authorizer = authorizer

	storageClient := storage.NewAccountsClient(subscription)
	storageClient.Authorizer = authorizer

	blobContainerClient := storage.NewBlobContainersClient(subscription)
	blobContainerClient.Authorizer = authorizer

	resultAccounts, err := storageClient.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, account := range resultAccounts.Values() {
			resourceGroup := &strings.Split(string(*account.ID), "/")[4]

			result, err := client.List(ctx, *resourceGroup, *account.Name, "", "", "")
			if err != nil {
				return nil, err
			}

			for {
				for _, v := range result.Values() {
					resourceGroup := strings.Split(*v.ID, "/")[4]
					accountName := strings.Split(*v.ID, "/")[8]

					op, err := blobContainerClient.GetImmutabilityPolicy(ctx, resourceGroup, accountName, *v.Name, "")
					if err != nil {
						return nil, err
					}

					values = append(values, Resource{
						ID:       *v.ID,
						Name:     *v.Name,
						Location: "global",
						Description: model.StorageContainerDescription{
							AccountName:        *account.Name,
							ListContainerItem:  v,
							ImmutabilityPolicy: op,
							ResourceGroup:      resourceGroup,
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

		if !resultAccounts.NotDone() {
			break
		}

		err = resultAccounts.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}
func StorageAccount(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	encryptionScopesStorageClient := storage.NewEncryptionScopesClient(subscription)
	encryptionScopesStorageClient.Authorizer = authorizer

	client := insights.NewDiagnosticSettingsClient(subscription)
	client.Authorizer = authorizer

	fileServicesStorageClient := storage.NewFileServicesClient(subscription)
	fileServicesStorageClient.Authorizer = authorizer

	blobServicesStorageClient := storage.NewBlobServicesClient(subscription)
	blobServicesStorageClient.Authorizer = authorizer

	managementPoliciesStorageClient := storage.NewManagementPoliciesClient(subscription)
	managementPoliciesStorageClient.Authorizer = authorizer

	storageClient := storage.NewAccountsClient(subscription)
	storageClient.Authorizer = authorizer

	result, err := storageClient.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, account := range result.Values() {
			resourceGroup := &strings.Split(*account.ID, "/")[4]

			var managementPolicy *storage.ManagementPolicy
			storageGetOp, err := managementPoliciesStorageClient.Get(ctx, *resourceGroup, *account.Name)
			if err != nil {
				if !strings.Contains(err.Error(), "ManagementPolicyNotFound") {
					return nil, err
				}
			} else {
				managementPolicy = &storageGetOp
			}

			var blobServicesProperties *storage.BlobServiceProperties
			if account.Kind != "FileStorage" {
				blobServicesPropertiesOp, err := blobServicesStorageClient.GetServiceProperties(ctx, *resourceGroup, *account.Name)
				if err != nil {
					return nil, err
				}
				blobServicesProperties = &blobServicesPropertiesOp
			}

			var logging *accounts.Logging
			if account.Kind != "FileStorage" {
				v, err := storageClient.ListKeys(ctx, *resourceGroup, *account.Name, "")
				if err != nil {
					if !strings.Contains(err.Error(), "ScopeLocked") {
						return nil, err
					}
				} else {
					if *v.Keys != nil || len(*v.Keys) > 0 {
						key := (*v.Keys)[0]

						storageAuth, err := autorest.NewSharedKeyAuthorizer(*account.Name, *key.Value, autorest.SharedKeyLite)
						if err != nil {
							return nil, err
						}

						client := accounts.New()
						client.Client.Authorizer = storageAuth
						client.BaseURI = storage.DefaultBaseURI

						resp, err := client.GetServiceProperties(ctx, *account.Name)
						if err != nil {
							if !strings.Contains(err.Error(), "FeatureNotSupportedForAccount") {
								return nil, err
							}
						} else {
							logging = resp.StorageServiceProperties.Logging
						}
					}
				}
			}

			var storageGetServicePropertiesOp *storage.FileServiceProperties
			if account.Kind != "BlobStorage" {
				v, err := fileServicesStorageClient.GetServiceProperties(ctx, *resourceGroup, *account.Name)
				if err != nil {
					if !strings.Contains(err.Error(), "FeatureNotSupportedForAccount") {
						return nil, err
					}
				}
				storageGetServicePropertiesOp = &v
			}

			diagSettingsOp, err := client.List(ctx, *account.ID)
			if err != nil {
				return nil, err
			}

			storageListEncryptionScope, err := encryptionScopesStorageClient.List(ctx, *resourceGroup, *account.Name)
			if err != nil {
				return nil, err
			}
			vsop := storageListEncryptionScope.Values()
			for storageListEncryptionScope.NotDone() {
				err := storageListEncryptionScope.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				vsop = append(vsop, storageListEncryptionScope.Values()...)
			}

			var storageProperties *queues.StorageServiceProperties
			if account.Sku.Tier == "Standard" && (account.Kind == "Storage" || account.Kind == "StorageV2") {
				accountKeys, err := storageClient.ListKeys(ctx, *resourceGroup, *account.Name, "")
				if err != nil {
					if !strings.Contains(err.Error(), "ScopeLocked") {
						return nil, err
					}
				} else {
					if *accountKeys.Keys != nil || len(*accountKeys.Keys) > 0 {
						key := (*accountKeys.Keys)[0]
						storageAuth, err := autorest.NewSharedKeyAuthorizer(*account.Name, *key.Value, autorest.SharedKeyLite)
						if err != nil {
							return nil, err
						}

						queuesClient := queues.New()
						queuesClient.Client.Authorizer = storageAuth
						queuesClient.BaseURI = storage.DefaultBaseURI

						resp, err := queuesClient.GetServiceProperties(ctx, *account.Name)

						if err != nil {
							if !strings.Contains(err.Error(), "FeatureNotSupportedForAccount") {
								return nil, err
							}
						} else {
							storageProperties = &resp.StorageServiceProperties
						}
					}
				}
			}

			values = append(values, Resource{
				ID:       *account.ID,
				Name:     *account.Name,
				Location: *account.Location,
				Description: model.StorageAccountDescription{
					Account:                     account,
					ManagementPolicy:            managementPolicy,
					BlobServiceProperties:       blobServicesProperties,
					Logging:                     logging,
					StorageServiceProperties:    storageProperties,
					FileServiceProperties:       storageGetServicePropertiesOp,
					DiagnosticSettingsResources: diagSettingsOp.Value,
					EncryptionScopes:            vsop,
					ResourceGroup:               *resourceGroup,
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
