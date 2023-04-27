package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/mariadb/mgmt/2020-01-01/mariadb"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func MariadbServer(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := mariadb.NewServersClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range *result.Value {
		resourceGroup := strings.Split(*v.ID, "/")[4]

		resource := Resource{
			ID:       *v.ID,
			Name:     *v.Name,
			Location: *v.Location,
			Description: model.MariadbServerDescription{
				Server:        v,
				ResourceGroup: resourceGroup,
			},
		}
		if stream != nil {
			if err := (*stream)(resource); err != nil {
				return nil, err
			}
		} else {
			values = append(values, resource)
		}
	}
	return values, nil
}
