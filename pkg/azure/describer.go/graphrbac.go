package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func AdUsers(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	graphClient := graphrbac.NewUsersClient(subscription) //TODO-Saleh tenant id ?
	graphClient.Authorizer = authorizer

	result, err := graphClient.List(ctx, "", "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, graph := range result.Values() {
			values = append(values, Resource{
				ID: *graph.ObjectID,
				Description: model.AdUsersDescription{
					AdUsers:                   graph,
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