package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/datafactory/mgmt/2018-06-01/datafactory"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func DataFactory(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	connClient := datafactory.NewPrivateEndPointConnectionsClient(subscription)
	connClient.Authorizer = authorizer
	factoryClient := datafactory.NewFactoriesClient(subscription)
	factoryClient.Authorizer = authorizer
	result, err := factoryClient.List(ctx)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for {
		for _, factory := range result.Values() {
			factoryName := factory.Name
			resourceGroup := strings.Split(*factory.ID, "/")[4]

			datafactoryListByFactoryOp, err := connClient.ListByFactory(ctx, resourceGroup, *factoryName)
			if err != nil {
				return nil, err
			}
			v := datafactoryListByFactoryOp.Values()
			for datafactoryListByFactoryOp.NotDone() {
				err := datafactoryListByFactoryOp.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				v = append(v, datafactoryListByFactoryOp.Values()...)
			}

			values = append(values, Resource{
				ID:       *factory.ID,
				Name:     *factory.Name,
				Location: *factory.Location,
				Description: model.DataFactoryDescription{
					Factory:                    factory,
					PrivateEndPointConnections: v,
					ResourceGroup:              resourceGroup,
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
