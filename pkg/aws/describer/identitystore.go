package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/identitystore"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func IdentityStoreGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := identitystore.NewFromConfig(cfg)
	paginator := identitystore.NewListGroupsPaginator(client, &identitystore.ListGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, group := range page.Groups {
			resource := Resource{
				ID:   *group.GroupId,
				Name: *group.DisplayName,
				Description: model.IdentityStoreGroupDescription{
					Group: group,
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
	}

	return values, nil
}

func IdentityStoreUser(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := identitystore.NewFromConfig(cfg)
	paginator := identitystore.NewListUsersPaginator(client, &identitystore.ListUsersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, user := range page.Users {
			resource := Resource{
				ID:   *user.UserId,
				Name: *user.UserName,
				Description: model.IdentityStoreUserDescription{
					User: user,
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
	}

	return values, nil
}
