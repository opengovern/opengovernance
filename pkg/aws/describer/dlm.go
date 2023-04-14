package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dlm"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func DLMLifecyclePolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	//describeCtx := GetDescribeContext(ctx)
	client := dlm.NewFromConfig(cfg)

	lifecyclePolicies, err := client.GetLifecyclePolicies(ctx, &dlm.GetLifecyclePoliciesInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, policySummary := range lifecyclePolicies.Policies {
		policy, err := client.GetLifecyclePolicy(ctx, &dlm.GetLifecyclePolicyInput{
			PolicyId: policySummary.PolicyId,
		})
		if err != nil {
			return nil, err
		}
		values = append(values, Resource{
			ID:  *policy.Policy.PolicyId,
			ARN: *policy.Policy.PolicyArn,
			Description: model.DLMLifecyclePolicyDescription{
				LifecyclePolicy: *policy.Policy,
			},
		})
	}

	return values, nil
}
