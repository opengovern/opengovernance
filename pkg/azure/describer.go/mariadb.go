package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/mariadb/mgmt/2020-01-01/mariadb"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func MariadbServer(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := mariadb.NewServersClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range *result.Value {
		values = append(values, Resource{
			ID: *v.ID,
			Description: model.MariadbServerDescription{
				v,
			},
		})
	}
	return values, nil
}
