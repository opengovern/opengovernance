package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2020-11-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func ContainerRegistry(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	containerRegistryClient := containerregistry.NewRegistriesClient(subscription)
	containerRegistryClient.Authorizer = authorizer

	client := containerregistry.NewRegistriesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, registry := range result.Values() {
			resourceGroup := strings.Split(*registry.ID, "/")[4]

			containerRegistryListCredentialsOp, err := containerRegistryClient.ListCredentials(ctx, resourceGroup, *registry.Name)
			if err != nil {
				if !strings.Contains(err.Error(), "UnAuthorizedForCredentialOperations") {
					return nil, err
				}
			}

			containerRegistryListUsagesOp, err := containerRegistryClient.ListUsages(ctx, resourceGroup, *registry.Name)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID: *registry.ID,
				Description: model.ContainerRegistryDescription{
					Registry:                      registry,
					RegistryListCredentialsResult: containerRegistryListCredentialsOp,
					RegistryUsages:                containerRegistryListUsagesOp.Value,
					ResourceGroup:                 resourceGroup,
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
