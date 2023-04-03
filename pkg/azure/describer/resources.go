package describer

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func listResourceGroups(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]resources.Group, error) {
	client := resources.NewGroupsClient(subscription)
	client.Authorizer = authorizer

	it, err := client.ListComplete(ctx, "", nil)
	if err != nil {
		return nil, err
	}

	var values []resources.Group
	for v := it.Value(); it.NotDone(); v = it.Value() {
		values = append(values, v)

		err := it.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func ResourceProvider(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := resources.NewProvidersClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx, nil, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, provider := range result.Values() {
			values = append(values, Resource{
				ID: *provider.ID,
				Description: model.ResourceProviderDescription{
					Provider: provider,
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

func ResourceGroup(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := resources.NewGroupsClient(subscription)
	client.Authorizer = authorizer

	groupListResultPage, err := client.List(ctx, "", nil)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, group := range groupListResultPage.Values() {
			values = append(values, Resource{
				ID:       *group.ID,
				Name:     *group.Name,
				Location: *group.Location,
				Description: model.ResourceGroupDescription{
					Group: group,
				},
			})
		}
		if !groupListResultPage.NotDone() {
			break
		}
		err = groupListResultPage.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}
