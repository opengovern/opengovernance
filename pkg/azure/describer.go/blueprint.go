package describer

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/blueprint/mgmt/blueprint"
	"github.com/Azure/go-autorest/autorest"
)

func BlueprintBlueprint(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]interface{}, error) {
	bps, err := blueprintBlueprint(ctx, authorizer, subscription)
	if err != nil {
		return nil, err
	}

	var values []interface{}
	for _, v := range bps {
		values = append(values, JSONAllFieldsMarshaller{Value: v})
	}

	return values, nil
}

func BlueprintArtifact(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]interface{}, error) {
	bps, err := blueprintBlueprint(ctx, authorizer, subscription)
	if err != nil {
		return nil, err
	}

	client := blueprint.NewArtifactsClient()
	client.Authorizer = authorizer

	var values []interface{}
	for _, bp := range bps {
		it, err := client.ListComplete(ctx, fmt.Sprintf("/subscriptions/%s", subscription), *bp.Name)
		if err != nil {
			return nil, err
		}

		for v := it.Value(); it.NotDone(); v = it.Value() {
			values = append(values, JSONAllFieldsMarshaller{Value: v})

			err := it.NextWithContext(ctx)
			if err != nil {
				return nil, err
			}
		}
	}

	return values, nil
}

func blueprintBlueprint(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]blueprint.Model, error) {
	client := blueprint.NewBlueprintsClient()
	client.Authorizer = authorizer

	it, err := client.ListComplete(ctx, fmt.Sprintf("/subscriptions/%s", subscription))
	if err != nil {
		return nil, err
	}

	var values []blueprint.Model
	for v := it.Value(); it.NotDone(); v = it.Value() {
		values = append(values, v)

		err := it.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}
