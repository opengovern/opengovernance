package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/identitystore"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func IdentityStoreGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := identitystore.NewFromConfig(cfg)
	paginator := identitystore.NewListGroupsPaginator(client, &identitystore.ListGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, group := range page.Groups {
			values = append(values, Resource{
				ID:   *group.GroupId,
				Name: *group.DisplayName,
				Description: model.IdentityStoreGroupDescription{
					Group: group,
				},
			})
		}
	}

	return values, nil
}

func IdentityStoreUser(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := identitystore.NewFromConfig(cfg)
	paginator := identitystore.NewListUsersPaginator(client, &identitystore.ListUsersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, user := range page.Users {
			values = append(values, Resource{
				ID:   *user.UserId,
				Name: *user.UserName,
				Description: model.IdentityStoreUserDescription{
					User: user,
				},
			})
		}
	}

	return values, nil
}
