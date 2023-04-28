package describer

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-06-01/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func Tenant(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := subscriptions.NewTenantsClient()
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range result.Values() {
		name := ""
		if v.DisplayName != nil {
			name = *v.DisplayName
		} else {
			name = *v.ID
		}
		resource := Resource{
			ID:       *v.ID,
			Name:     name,
			Location: "global",
			Description: model.TenantDescription{
				TenantIDDescription: v,
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

func Subscription(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := subscriptions.NewClient()
	client.Authorizer = authorizer

	op, err := client.Get(ctx, subscription)
	if err != nil {
		return nil, err
	}

	var values []Resource
	resource := Resource{
		ID:       *op.ID,
		Name:     *op.DisplayName,
		Location: "global",
		Description: model.SubscriptionDescription{
			Subscription: op,
		},
	}
	if stream != nil {
		if err := (*stream)(resource); err != nil {
			return nil, err
		}
	} else {
		values = append(values, resource)
	}

	return values, nil
}
