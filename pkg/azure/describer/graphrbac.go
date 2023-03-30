package describer

import (
	"context"
	"strings"

	"github.com/manicminer/hamilton/auth"
	"github.com/manicminer/hamilton/msgraph"
	"github.com/manicminer/hamilton/odata"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func AdUsers(ctx context.Context, authorizer auth.Authorizer, tenantId string) ([]Resource, error) {
	client := msgraph.NewUsersClient(tenantId)
	client.BaseClient.Authorizer = authorizer

	input := odata.Query{}
	input.Expand = odata.Expand{
		Relationship: "memberOf",
		Select:       []string{"id", "displayName"},
	}

	users, _, err := client.List(ctx, input)
	if err != nil {
		if strings.Contains(err.Error(), "Request_ResourceNotFound") {
			return nil, nil
		}
		return nil, err
	}

	var values []Resource
	for _, user := range *users {

		values = append(values, Resource{
			ID:       *user.ID,
			Name:     *user.DisplayName,
			Location: "global",
			Description: model.AdUsersDescription{
				TenantID: tenantId,
				AdUsers:  user,
			},
		})
	}

	return values, nil
}

func AdGroup(ctx context.Context, authorizer auth.Authorizer, tenantId string) ([]Resource, error) {
	client := msgraph.NewGroupsClient(tenantId)
	client.BaseClient.Authorizer = authorizer

	input := odata.Query{}

	groups, _, err := client.List(ctx, input)
	if err != nil {
		if strings.Contains(err.Error(), "Request_ResourceNotFound") {
			return nil, nil
		}
		return nil, err
	}

	var values []Resource
	for _, group := range *groups {
		values = append(values, Resource{
			ID:       *group.ID,
			Name:     *group.DisplayName,
			Location: "global",
			Description: model.AdGroupDescription{
				TenantID: tenantId,
				AdGroup:  group,
			},
		})
	}

	return values, nil
}

func AdServicePrinciple(ctx context.Context, authorizer auth.Authorizer, tenantId string) ([]Resource, error) {
	client := msgraph.NewServicePrincipalsClient(tenantId)
	client.BaseClient.Authorizer = authorizer

	input := odata.Query{}

	servicePrincipals, _, err := client.List(ctx, input)
	if err != nil {
		if strings.Contains(err.Error(), "Request_ResourceNotFound") {
			return nil, nil
		}
		return nil, err
	}

	var values []Resource
	for _, servicePrincipal := range *servicePrincipals {
		values = append(values, Resource{
			ID:       *servicePrincipal.ID,
			Name:     *servicePrincipal.DisplayName,
			Location: "global",
			Description: model.AdServicePrincipalDescription{
				TenantID:           tenantId,
				AdServicePrincipal: servicePrincipal,
			},
		})
	}

	return values, nil
}
