package describer

import (
	"context"
	"github.com/manicminer/hamilton/auth"
	"github.com/manicminer/hamilton/msgraph"
	"github.com/manicminer/hamilton/odata"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
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
			ID: *user.ID,
			Description: model.AdUsersDescription{
				AdUsers: user,
			},
		})
	}

	return values, nil
}
