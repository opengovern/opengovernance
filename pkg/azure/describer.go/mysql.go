package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/mysql/mgmt/2020-01-01/mysql"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func MysqlServer(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	keysClient := mysql.NewServerKeysClient(subscription)
	keysClient.Authorizer = authorizer

	mysqlClient := mysql.NewConfigurationsClient(subscription)
	mysqlClient.Authorizer = authorizer

	client := mysql.NewServersClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, server := range *result.Value {
		resourceGroup := strings.Split(string(*server.ID), "/")[4]
		serverName := *server.Name

		mysqlListByServerOp, err := mysqlClient.ListByServer(ctx, resourceGroup, serverName)
		if err != nil {
			return nil, err
		}

		keysListOp, err := keysClient.List(ctx, resourceGroup, serverName)
		if err != nil {
			return nil, err
		}

		var keys []mysql.ServerKey
		keys = append(keys, keysListOp.Values()...)
		for keysListOp.NotDone() {
			err = keysListOp.NextWithContext(ctx)
			if err != nil {
				return nil, err
			}
			keys = append(keys, keysListOp.Values()...)
		}

		values = append(values, Resource{
			ID:       *server.ID,
			Location: *server.Location,
			Description: model.MysqlServerDescription{
				Server:         server,
				Configurations: mysqlListByServerOp.Value,
				ServerKeys:     keys,
				ResourceGroup:  resourceGroup,
			},
		})
	}
	return values, nil
}
