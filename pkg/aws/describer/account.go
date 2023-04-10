package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/account"
	"github.com/aws/aws-sdk-go-v2/service/account/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func AccountAlternateContact(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := account.NewFromConfig(cfg)

	var values []Resource

	contactTypes := []string{"BILLING", "OPERATIONS", "SECURITY"}
	input := &account.GetAlternateContactInput{
		AccountId: &describeCtx.AccountID,
	}
	for _, contactType := range contactTypes {
		input.AlternateContactType = types.AlternateContactType(contactType)
		op, err := client.GetAlternateContact(ctx, input)
		if err != nil {
			if isErr(err, "ResourceNotFoundException") {
				continue
			}
			return nil, err
		}

		values = append(values, Resource{
			Name: *op.AlternateContact.Name,
			Description: model.AccountAlternateContactDescription{
				AlternateContact: *op.AlternateContact,
				LinkedAccountID:  describeCtx.AccountID,
			},
		})
	}

	return values, nil
}

func AccountContact(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := account.NewFromConfig(cfg)

	var values []Resource

	input := &account.GetContactInformationInput{
		AccountId: &describeCtx.AccountID,
	}
	op, err := client.GetContactInformation(ctx, input)
	if err != nil {
		return nil, err
	}

	values = append(values, Resource{
		Name: *op.ContactInformation.FullName,
		Description: model.AccountContactDescription{
			AlternateContact: *op.ContactInformation,
			LinkedAccountID:  describeCtx.AccountID,
		},
	})

	return values, nil
}
