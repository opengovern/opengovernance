package describer

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/blueprint/mgmt/blueprint"
	"github.com/Azure/go-autorest/autorest"
)

func BlueprintBlueprint(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	bps, err := blueprintBlueprint(ctx, authorizer, subscription)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range bps {
		resource := Resource{
			ID:          *v.ID,
			Description: JSONAllFieldsMarshaller{Value: v},
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

func BlueprintArtifact(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	bps, err := blueprintBlueprint(ctx, authorizer, subscription)
	if err != nil {
		return nil, err
	}

	client := blueprint.NewArtifactsClient()
	client.Authorizer = authorizer

	var values []Resource
	for _, bp := range bps {
		it, err := client.ListComplete(ctx, fmt.Sprintf("/subscriptions/%s", subscription), *bp.Name)
		if err != nil {
			return nil, err
		}

		for v := it.Value(); it.NotDone(); v = it.Value() {
			var (
				id    string
				value interface{}
			)
			if artifact, ok := v.AsArtifact(); ok {
				id, value = *artifact.ID, artifact
			} else if artifact, ok := v.AsTemplateArtifact(); ok {
				id, value = *artifact.ID, artifact
			} else if artifact, ok := v.AsPolicyAssignmentArtifact(); ok {
				id, value = *artifact.ID, artifact
			} else if artifact, ok := v.AsRoleAssignmentArtifact(); ok {
				id, value = *artifact.ID, artifact
			} else {
				panic("unknown artifact type")
			}

			resource := Resource{
				ID:          id,
				Description: JSONAllFieldsMarshaller{Value: value},
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
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
