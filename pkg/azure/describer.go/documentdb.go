package describer

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/cosmos-db/mgmt/documentdb"
	"github.com/Azure/go-autorest/autorest"
)

func DocumentDBDatabaseAccountsSQLDatabase(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]interface{}, error) {
	rgs, err := resourceGroup(ctx, authorizer, subscription)
	if err != nil {
		return nil, err
	}

	client := documentdb.NewSQLResourcesClient(subscription)
	client.Authorizer = authorizer

	var values []interface{}
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
				values = append(values, JSONAllFieldsMarshaller{Value: v})
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
