package describer

import (
	"context"
	"strings"

	sub "github.com/Azure/azure-sdk-for-go/profiles/latest/subscription/mgmt/subscription"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func Location(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	subscriptionsClient := sub.NewSubscriptionsClient()
	subscriptionsClient.Authorizer = authorizer

	result, err := subscriptionsClient.ListLocations(ctx, subscription)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, location := range *result.Value {
		resourceGroup := strings.Split(*location.ID, "/")[4]

		resource := Resource{
			ID:       *location.ID,
			Name:     *location.Name,
			Location: "global",
			Description: model.LocationDescription{
				Location:      location,
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
