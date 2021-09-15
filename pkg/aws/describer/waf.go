package describer

/*

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafregional"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
)

func WAFv2IPSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafv2.NewFromConfig(cfg)
	paginator := wafv2.NewDescribeIPSetsPaginator(client, &wafv2.DescribeIPSetsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.IPSets {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFv2LoggingConfiguration(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafv2.NewFromConfig(cfg)
	paginator := wafv2.NewDescribeLoggingConfigurationsPaginator(client, &wafv2.DescribeLoggingConfigurationsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.LoggingConfigurations {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFv2RegexPatternSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafv2.NewFromConfig(cfg)
	paginator := wafv2.NewDescribeRegexPatternSetsPaginator(client, &wafv2.DescribeRegexPatternSetsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.RegexPatternSets {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFv2RuleGroup(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafv2.NewFromConfig(cfg)
	paginator := wafv2.NewDescribeRuleGroupsPaginator(client, &wafv2.DescribeRuleGroupsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.RuleGroups {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFv2WebACL(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafv2.NewFromConfig(cfg)
	paginator := wafv2.NewDescribeWebACLsPaginator(client, &wafv2.DescribeWebACLsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.WebACLs {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFv2WebACLAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafv2.NewFromConfig(cfg)
	paginator := wafv2.NewDescribeWebACLAssociationsPaginator(client, &wafv2.DescribeWebACLAssociationsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.WebACLAssociations {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFRegionalByteMatchSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafregional.NewFromConfig(cfg)
	paginator := wafregional.NewDescribeByteMatchSetsPaginator(client, &wafregional.DescribeByteMatchSetsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ByteMatchSets {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFRegionalGeoMatchSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafregional.NewFromConfig(cfg)
	paginator := wafregional.NewDescribeGeoMatchSetsPaginator(client, &wafregional.DescribeGeoMatchSetsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.GeoMatchSets {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFRegionalIPSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafregional.NewFromConfig(cfg)
	paginator := client.ListIPSets(&wafregional.ListIPSetsInput{Limit: pagingSize})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.IPSets {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFRegionalRateBasedRule(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafregional.NewFromConfig(cfg)
	paginator := wafregional.NewDescribeRateBasedRulesPaginator(client, &wafregional.DescribeRateBasedRulesInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.RateBasedRules {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFRegionalRegexPatternSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafregional.NewFromConfig(cfg)
	paginator := wafregional.NewDescribeRegexPatternSetsPaginator(client, &wafregional.DescribeRegexPatternSetsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.RegexPatternSets {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFRegionalRule(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafregional.NewFromConfig(cfg)
	paginator := wafregional.NewDescribeRulesPaginator(client, &wafregional.DescribeRulesInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Rules {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFRegionalSizeConstraintSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafregional.NewFromConfig(cfg)
	paginator := wafregional.NewDescribeSizeConstraintSetsPaginator(client, &wafregional.DescribeSizeConstraintSetsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.SizeConstraintSets {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFRegionalSqlInjectionMatchSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafregional.NewFromConfig(cfg)
	paginator := wafregional.NewDescribeSqlInjectionMatchSetsPaginator(client, &wafregional.DescribeSqlInjectionMatchSetsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.SqlInjectionMatchSets {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFRegionalWebACL(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafregional.NewFromConfig(cfg)
	paginator := wafregional.NewDescribeWebACLsPaginator(client, &wafregional.DescribeWebACLsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.WebACLs {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFRegionalWebACLAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafregional.NewFromConfig(cfg)
	paginator := wafregional.NewDescribeWebACLAssociationsPaginator(client, &wafregional.DescribeWebACLAssociationsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.WebACLAssociations {
			values = append(values, v)
		}
	}

	return values, nil
}

func WAFRegionalXssMatchSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := wafregional.NewFromConfig(cfg)
	paginator := wafregional.NewDescribeXssMatchSetsPaginator(client, &wafregional.DescribeXssMatchSetsInput{MaxResults: aws.Int32(pagingSize)})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.XssMatchSets {
			values = append(values, v)
		}
	}

	return values, nil
}
*/