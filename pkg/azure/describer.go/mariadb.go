package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/mariadb/mgmt/2020-01-01/mariadb"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
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
		resourceGroup := strings.Split(*v.ID, "/")[4]

		values = append(values, Resource{
			ID:       *v.ID,
			Name:     *v.Name,
			Location: *v.Location,
			Description: model.MariadbServerDescription{
				Server:        v,
				ResourceGroup: resourceGroup,
			},
		})
	}
	return values, nil
}
