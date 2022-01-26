package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/links"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

//TODO-Saleh resource ??
func ResourceLink(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := links.NewResourceLinksClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListAtSubscription(ctx, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			values = append(values, Resource{
				ID: *v.ID,
				Description: model.ResourceLinkDescription{
					ResourceLink: v,
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
