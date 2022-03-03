package describer

import (
	"context"
	sub "github.com/Azure/azure-sdk-for-go/profiles/latest/subscription/mgmt/subscription"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func Location(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	subscriptionsClient := sub.NewSubscriptionsClient()
	subscriptionsClient.Authorizer = authorizer

	result, err := subscriptionsClient.ListLocations(ctx, subscription)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, location := range *result.Value {
		resourceGroup := strings.Split(*location.ID, "/")[4]

		values = append(values, Resource{
			ID:       *location.ID,
			Location: "global",
			Description: model.LocationDescription{
				Location:      location,
				ResourceGroup: resourceGroup,
			},
		})
	}
	return values, nil
}
