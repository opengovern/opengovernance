package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/preview/cosmos-db/mgmt/2020-04-01-preview/documentdb"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func DocumentDBDatabaseAccountsSQLDatabase(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	rgs, err := resourceGroup(ctx, authorizer, subscription)
	if err != nil {
		return nil, err
	}

	client := documentdb.NewSQLResourcesClient(subscription)
	client.Authorizer = authorizer

	var values []Resource
	for _, rg := range rgs {
		accounts, err := documentDBDatabaseAccounts(ctx, authorizer, subscription, *rg.Name)
		if err != nil {
			return nil, err
		}

		for _, account := range accounts {
			it, err := client.ListSQLDatabases(ctx, *rg.Name, *account.Name)
			if err != nil {
				return nil, err
			} else if it.Value == nil {
				continue
			}

			for _, v := range *it.Value {
				values = append(values, Resource{
					ID:          *v.ID,
					Location:    *v.Location,
					Description: JSONAllFieldsMarshaller{Value: v},
				})
			}
		}
	}

	return values, nil

}

func documentDBDatabaseAccounts(ctx context.Context, authorizer autorest.Authorizer, subscription string, resourceGroup string) ([]documentdb.DatabaseAccountGetResults, error) {
	client := documentdb.NewDatabaseAccountsClient(subscription)
	client.Authorizer = authorizer

	accounts, err := client.ListByResourceGroup(ctx, resourceGroup)
	if err != nil {
		return nil, err
	} else if accounts.Value == nil {
		return nil, nil
	}

	var values []documentdb.DatabaseAccountGetResults
	values = append(values, *accounts.Value...)

	return values, nil
}

func CosmosdbAccount(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	documentDBClient := documentdb.NewDatabaseAccountsClient(subscription)
	documentDBClient.Authorizer = authorizer
	result, err := documentDBClient.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource

	for _, account := range *result.Value {
		resourceGroup := strings.Split(*account.ID, "/")[4]

		values = append(values, Resource{
			ID:       *account.ID,
			Location: *account.Location,
			Description: model.CosmosdbAccountDescription{
				DatabaseAccountGetResults: account,
				ResourceGroup:             resourceGroup,
			},
		})
	}
	return values, nil
}
