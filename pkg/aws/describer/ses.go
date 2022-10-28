package describer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func SESConfigurationSet(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := sesv2.NewFromConfig(cfg)
	paginator := sesv2.NewListConfigurationSetsPaginator(client, &sesv2.ListConfigurationSetsInput{})

	sesClient := ses.NewFromConfig(cfg)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ConfigurationSets {
			output, err := sesClient.DescribeConfigurationSet(ctx, &ses.DescribeConfigurationSetInput{ConfigurationSetName: aws.String(v)})
			if err != nil {
				return nil, err
			}

			arn := fmt.Sprintf("arn:%s:ses:%s:%s:configuration-set/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *output.ConfigurationSet.Name)

			values = append(values, Resource{
				ARN:  arn,
				Name: *output.ConfigurationSet.Name,
				Description: model.SESConfigurationSetDescription{
					ConfigurationSet: *output.ConfigurationSet,
				},
			})
		}
	}

	return values, nil
}

func SESIdentity(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := sesv2.NewFromConfig(cfg)
	paginator := sesv2.NewListEmailIdentitiesPaginator(client, &sesv2.ListEmailIdentitiesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.EmailIdentities {
			arn := fmt.Sprintf("arn:%s:ses:%s:%s:identity/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *v.IdentityName)

			tags, err := client.ListTagsForResource(ctx, &sesv2.ListTagsForResourceInput{
				ResourceArn: &arn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  arn,
				Name: *v.IdentityName,
				Description: model.SESIdentityDescription{
					Identity: v,
					Tags:     tags.Tags,
				},
			})
		}
	}

	return values, nil
}

func SESContactList(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := sesv2.NewFromConfig(cfg)
	paginator := sesv2.NewListContactListsPaginator(client, &sesv2.ListContactListsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ContactLists {
			values = append(values, Resource{
				ID:          *v.ContactListName,
				Name:        *v.ContactListName,
				Description: v,
			})
		}
	}

	return values, nil
}

func SESReceiptFilter(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ses.NewFromConfig(cfg)

	output, err := client.ListReceiptFilters(ctx, &ses.ListReceiptFiltersInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.Filters {
		values = append(values, Resource{
			ID:          *v.Name,
			Name:        *v.Name,
			Description: v,
		})
	}

	return values, nil
}

func SESReceiptRuleSet(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ses.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.ListReceiptRuleSets(ctx, &ses.ListReceiptRuleSetsInput{NextToken: prevToken})
		if err != nil {
			return nil, err
		}

		for _, v := range output.RuleSets {
			output, err := client.DescribeReceiptRuleSet(ctx, &ses.DescribeReceiptRuleSetInput{RuleSetName: v.Name})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:          *output.Metadata.Name,
				Name:        *output.Metadata.Name,
				Description: output,
			})
		}

		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func SESTemplate(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ses.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.ListTemplates(ctx, &ses.ListTemplatesInput{NextToken: prevToken})
		if err != nil {
			return nil, err
		}

		for _, v := range output.TemplatesMetadata {
			values = append(values, Resource{
				ID:          *v.Name,
				Name:        *v.Name,
				Description: v,
			})
		}

		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}
