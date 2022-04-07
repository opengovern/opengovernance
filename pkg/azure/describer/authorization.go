package describer

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func RoleAssignment(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := authorization.NewRoleAssignmentsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			values = append(values, Resource{
				ID:       *v.ID,
				Name:     *v.Name,
				Location: "global",
				Description: model.RoleAssignmentDescription{
					RoleAssignment: v,
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

func RoleDefinition(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := authorization.NewRoleDefinitionsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx, "/subscriptions/"+subscription, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			values = append(values, Resource{
				ID:       *v.ID,
				Name:     *v.Name,
				Location: "global",
				Description: model.RoleDefinitionDescription{
					RoleDefinition: v,
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
