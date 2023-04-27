package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-09-01/policy"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func RoleAssignment(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := authorization.NewRoleAssignmentsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			resource := Resource{
				ID:       *v.ID,
				Name:     *v.Name,
				Location: "global",
				Description: model.RoleAssignmentDescription{
					RoleAssignment: v,
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

func RoleDefinition(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := authorization.NewRoleDefinitionsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx, "/subscriptions/"+subscription, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			resource := Resource{
				ID:       *v.ID,
				Name:     *v.Name,
				Location: "global",
				Description: model.RoleDefinitionDescription{
					RoleDefinition: v,
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

func PolicyDefinition(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := policy.NewDefinitionsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, definition := range result.Values() {
			akas := []string{"azure:///subscriptions/" + subscription + *definition.ID, "azure:///subscriptions/" + subscription + strings.ToLower(*definition.ID)}
			turbotData := map[string]interface{}{
				"SubscriptionId": subscription,
				"Akas":           akas,
			}

			resource := Resource{
				ID:       *definition.ID,
				Name:     *definition.Name,
				Location: "global",
				Description: model.PolicyDefinitionDescription{
					Definition: definition,
					TurboData:  turbotData,
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
