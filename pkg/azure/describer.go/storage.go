package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

//TODO-Saleh resource??
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
						ID: *v.ID,
						Description: JSONAllFieldsMarshaller{
							model.StorageContainerDescription{
								ListContainerItem:  v,
								ImmutabilityPolicy: op,
							},
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
