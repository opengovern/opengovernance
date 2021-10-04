package describer

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/go-autorest/autorest"
)

func resourceGroup(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]resources.Group, error) {
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
