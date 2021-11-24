package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

func SESConfigurationSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := sesv2.NewFromConfig(cfg)
	paginator := sesv2.NewListConfigurationSetsPaginator(client, &sesv2.ListConfigurationSetsInput{})

	sesClient := ses.NewFromConfig(cfg)

	var values []interface{}
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

			values = append(values, output)
		}
	}

	return values, nil
}

func SESContactList(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := sesv2.NewFromConfig(cfg)
	paginator := sesv2.NewListContactListsPaginator(client, &sesv2.ListContactListsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ContactLists {
			values = append(values, v)
		}
	}

	return values, nil
}

func SESReceiptFilter(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ses.NewFromConfig(cfg)

	output, err := client.ListReceiptFilters(ctx, &ses.ListReceiptFiltersInput{})
	if err != nil {
		return nil, err
	}

	var values []interface{}
	for _, v := range output.Filters {
		values = append(values, v)
	}

	return values, nil
}

func SESReceiptRuleSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ses.NewFromConfig(cfg)

	var values []interface{}
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

			values = append(values, output)
		}

		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func SESTemplate(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ses.NewFromConfig(cfg)

	var values []interface{}
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.ListTemplates(ctx, &ses.ListTemplatesInput{NextToken: prevToken})
		if err != nil {
			return nil, err
		}

		for _, v := range output.TemplatesMetadata {
			values = append(values, v)
		}

		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}
