package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-06-01/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

//TODO-Saleh resource ??
func Tenant(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := subscriptions.NewTenantsClient()
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range result.Values() {
		values = append(values, Resource{
			ID: *v.ID,
			Description: JSONAllFieldsMarshaller{
				model.TenantDescription{
					TenantIDDescription: v,
				},
			},
		})
	}

	return values, nil
}
