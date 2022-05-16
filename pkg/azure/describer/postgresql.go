package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/postgresql/mgmt/2020-01-01/postgresql"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func PostgresqlServer(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	firewallClient := postgresql.NewFirewallRulesClient(subscription)
	firewallClient.Authorizer = authorizer
	keysClient := postgresql.NewServerKeysClient(subscription)
	keysClient.Authorizer = authorizer
	confClient := postgresql.NewConfigurationsClient(subscription)
	confClient.Authorizer = authorizer
	adminClient := postgresql.NewServerAdministratorsClient(subscription)
	adminClient.Authorizer = authorizer
	client := postgresql.NewServersClient(subscription)
	client.Authorizer = authorizer
	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for _, server := range *result.Value {
		resourceGroupName := strings.Split(string(*server.ID), "/")[4]
		adminListOp, err := adminClient.List(ctx, resourceGroupName, *server.Name)
		if err != nil {
			return nil, err
		}

		confListByServerOp, err := confClient.ListByServer(ctx, resourceGroupName, *server.Name)
		if err != nil {
			return nil, err
		}

		keysListOp, err := keysClient.List(ctx, resourceGroupName, *server.Name)
		if err != nil {
			return nil, err
		}
		kop := keysListOp.Values()
		for keysListOp.NotDone() {
			err := keysListOp.NextWithContext(ctx)
			if err != nil {
				return nil, err
			}

			kop = append(kop, keysListOp.Values()...)
		}

		firewallListByServerOp, err := firewallClient.ListByServer(ctx, resourceGroupName, *server.Name)
		if err != nil {
			return nil, err
		}

		values = append(values, Resource{
			ID:       *server.ID,
			Name:     *server.Name,
			Location: *server.Location,
			Description: model.PostgresqlServerDescription{
				Server:                       server,
				ServerAdministratorResources: adminListOp.Value,
				Configurations:               confListByServerOp.Value,
				ServerKeys:                   kop,
				FirewallRules:                firewallListByServerOp.Value,
				ResourceGroup:                resourceGroupName,
			},
		})
	}
	return values, nil
}
