package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func EventBridgeBus(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := eventbridge.NewFromConfig(cfg)

	input := eventbridge.ListEventBusesInput{Limit: aws.Int32(100)}

	var values []Resource
	for {
		response, err := client.ListEventBuses(ctx, &input)
		if err != nil {
			if isErr(err, "InvalidParameter") || isErr(err, "ResourceNotFoundException") || isErr(err, "ValidationException") {
				return nil, nil
			}
			return nil, err
		}
		for _, bus := range response.EventBuses {
			tagsOutput, err := client.ListTagsForResource(ctx, &eventbridge.ListTagsForResourceInput{
				ResourceARN: bus.Arn,
			})
			if err != nil {
				if !isErr(err, "InvalidParameter") && !isErr(err, "ResourceNotFoundException") && !isErr(err, "ValidationException") {
					return nil, err
				}
				tagsOutput = &eventbridge.ListTagsForResourceOutput{}
			}

			values = append(values, Resource{
				ARN:  *bus.Arn,
				Name: *bus.Name,
				Description: model.EventBridgeBusDescription{
					Bus:  bus,
					Tags: tagsOutput.Tags,
				},
			})
		}
		if response.NextToken == nil {
			break
		}
		input.NextToken = response.NextToken
	}

	return values, nil
}

func EventBridgeRule(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := eventbridge.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		listRulesOutput, err := client.ListRules(ctx, &eventbridge.ListRulesInput{
			NextToken: prevToken,
		})
		if err != nil {
			return nil, err
		}
		for _, listRule := range listRulesOutput.Rules {
			rule, err := client.DescribeRule(ctx, &eventbridge.DescribeRuleInput{
				Name: listRule.Name,
			})
			if err != nil {
				return nil, err
			}

			tagsOutput, err := client.ListTagsForResource(ctx, &eventbridge.ListTagsForResourceInput{
				ResourceARN: rule.Arn,
			})
			if err != nil {
				if !isErr(err, "ResourceNotFoundException") && !isErr(err, "ValidationException") {
					return nil, err
				}
				tagsOutput = &eventbridge.ListTagsForResourceOutput{}
			}

			targets, err := client.ListTargetsByRule(ctx, &eventbridge.ListTargetsByRuleInput{
				Rule: listRule.Name,
			})
			if err != nil {
				if !isErr(err, "ResourceNotFoundException") && !isErr(err, "ValidationException") {
					return nil, err
				}
				targets = &eventbridge.ListTargetsByRuleOutput{}
			}

			values = append(values, Resource{
				ARN:  *rule.Arn,
				Name: *rule.Name,
				Description: model.EventBridgeRuleDescription{
					Rule:    *rule,
					Tags:    tagsOutput.Tags,
					Targets: targets.Targets,
				},
			})
		}

		return listRulesOutput.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}
