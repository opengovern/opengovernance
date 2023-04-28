package describer

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/policy"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func PolicyAssignment(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := policy.NewAssignmentsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			location := ""
			if v.Location != nil {
				location = *v.Location
			}

			resource := Resource{
				ID:       *v.ID,
				Name:     *v.Name,
				Location: location,
				Description: model.PolicyAssignmentDescription{
					Assignment: v,
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
