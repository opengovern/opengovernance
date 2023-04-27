package describer

import (
	"context"
	"errors"

	"github.com/aws/smithy-go"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/turbot/go-kit/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func KMSAlias(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := kms.NewFromConfig(cfg)
	paginator := kms.NewListAliasesPaginator(client, &kms.ListAliasesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Aliases {
			resource := Resource{
				ARN:         *v.AliasArn,
				Name:        *v.AliasName,
				Description: v,
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

func KMSKey(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := kms.NewFromConfig(cfg)
	paginator := kms.NewListKeysPaginator(client, &kms.ListKeysInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Keys {
			key, err := client.DescribeKey(ctx, &kms.DescribeKeyInput{
				KeyId: v.KeyId,
			})
			if err != nil {
				return nil, err
			}

			aliasPaginator := kms.NewListAliasesPaginator(client, &kms.ListAliasesInput{
				KeyId: v.KeyId,
			})

			var keyAlias []types.AliasListEntry
			for aliasPaginator.HasMorePages() {
				aliasPage, err := aliasPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}

				keyAlias = append(keyAlias, aliasPage.Aliases...)
			}

			rotationStatus, err := client.GetKeyRotationStatus(ctx, &kms.GetKeyRotationStatusInput{
				KeyId: v.KeyId,
			})
			if err != nil {
				// For AWS managed KMS keys GetKeyRotationStatus API generates exceptions
				var ae smithy.APIError
				if errors.As(err, &ae) &&
					helpers.StringSliceContains([]string{"AccessDeniedException", "UnsupportedOperationException"}, ae.ErrorCode()) {
					rotationStatus = &kms.GetKeyRotationStatusOutput{}
					err = nil
				}

				if a, ok := err.(awserr.Error); ok {
					if helpers.StringSliceContains([]string{"AccessDeniedException", "UnsupportedOperationException"}, a.Code()) {
						rotationStatus = &kms.GetKeyRotationStatusOutput{}
						err = nil
					}
				}

				if err != nil {
					return nil, err
				}
			}

			var defaultPolicy = "default"
			policy, err := client.GetKeyPolicy(ctx, &kms.GetKeyPolicyInput{
				KeyId:      v.KeyId,
				PolicyName: &defaultPolicy,
			})
			if err != nil {
				return nil, err
			}

			tags, err := client.ListResourceTags(ctx, &kms.ListResourceTagsInput{
				KeyId: v.KeyId,
			})
			if err != nil {
				if isErr(err, "AccessDeniedException") {
					tags = &kms.ListResourceTagsOutput{}
				} else {
					return nil, err
				}
			}

			resource := Resource{
				ARN:  *v.KeyArn,
				Name: *v.KeyId,
				Description: model.KMSKeyDescription{
					Metadata:           key.KeyMetadata,
					Aliases:            keyAlias,
					KeyRotationEnabled: rotationStatus.KeyRotationEnabled,
					Policy:             policy.Policy,
					Tags:               tags.Tags,
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

func GetKMSKey(ctx context.Context, cfg aws.Config, fields map[string]string) ([]Resource, error) {
	id := fields["id"]
	client := kms.NewFromConfig(cfg)

	var values []Resource
	key, err := client.DescribeKey(ctx, &kms.DescribeKeyInput{
		KeyId: &id,
	})
	if err != nil {
		return nil, err
	}
	v := key.KeyMetadata

	aliasPaginator := kms.NewListAliasesPaginator(client, &kms.ListAliasesInput{
		KeyId: v.KeyId,
	})

	var keyAlias []types.AliasListEntry
	for aliasPaginator.HasMorePages() {
		aliasPage, err := aliasPaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		keyAlias = append(keyAlias, aliasPage.Aliases...)
	}

	rotationStatus, err := client.GetKeyRotationStatus(ctx, &kms.GetKeyRotationStatusInput{
		KeyId: v.KeyId,
	})
	if err != nil {
		// For AWS managed KMS keys GetKeyRotationStatus API generates exceptions
		var ae smithy.APIError
		if errors.As(err, &ae) &&
			helpers.StringSliceContains([]string{"AccessDeniedException", "UnsupportedOperationException"}, ae.ErrorCode()) {
			rotationStatus = &kms.GetKeyRotationStatusOutput{}
			err = nil
		}

		if a, ok := err.(awserr.Error); ok {
			if helpers.StringSliceContains([]string{"AccessDeniedException", "UnsupportedOperationException"}, a.Code()) {
				rotationStatus = &kms.GetKeyRotationStatusOutput{}
				err = nil
			}
		}

		if err != nil {
			return nil, err
		}
	}

	var defaultPolicy = "default"
	policy, err := client.GetKeyPolicy(ctx, &kms.GetKeyPolicyInput{
		KeyId:      v.KeyId,
		PolicyName: &defaultPolicy,
	})
	if err != nil {
		return nil, err
	}

	tags, err := client.ListResourceTags(ctx, &kms.ListResourceTagsInput{
		KeyId: v.KeyId,
	})
	if err != nil {
		if isErr(err, "AccessDeniedException") {
			tags = &kms.ListResourceTagsOutput{}
		} else {
			return nil, err
		}
	}

	values = append(values, Resource{
		ARN:  *key.KeyMetadata.Arn,
		Name: *v.KeyId,
		Description: model.KMSKeyDescription{
			Metadata:           key.KeyMetadata,
			Aliases:            keyAlias,
			KeyRotationEnabled: rotationStatus.KeyRotationEnabled,
			Policy:             policy.Policy,
			Tags:               tags.Tags,
		},
	})

	return values, nil
}
