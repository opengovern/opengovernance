package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/glacier"
	"github.com/aws/aws-sdk-go-v2/service/glacier/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func GlacierVault(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := glacier.NewFromConfig(cfg)
	paginator := glacier.NewListVaultsPaginator(client, &glacier.ListVaultsInput{
		AccountId: &describeCtx.AccountID,
	})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !isErr(err, "ResourceNotFoundException") && !isErr(err, "InvalidParameter") {
				return nil, err
			}
			continue
		}

		for _, vault := range page.VaultList {
			accessPolicy, err := client.GetVaultAccessPolicy(ctx, &glacier.GetVaultAccessPolicyInput{
				AccountId: &describeCtx.AccountID,
				VaultName: vault.VaultName,
			})
			if err != nil {
				if !isErr(err, "ResourceNotFoundException") && !isErr(err, "InvalidParameter") {
					return nil, err
				}
				accessPolicy = &glacier.GetVaultAccessPolicyOutput{
					Policy: &types.VaultAccessPolicy{},
				}
			}

			lockPolicy, err := client.GetVaultLock(ctx, &glacier.GetVaultLockInput{
				AccountId: &describeCtx.AccountID,
				VaultName: vault.VaultName,
			})
			if err != nil {
				if !isErr(err, "ResourceNotFoundException") && !isErr(err, "InvalidParameter") {
					return nil, err
				}
				lockPolicy = &glacier.GetVaultLockOutput{}
			}

			tags, err := client.ListTagsForVault(ctx, &glacier.ListTagsForVaultInput{
				AccountId: &describeCtx.AccountID,
				VaultName: vault.VaultName,
			})
			if err != nil {
				if !isErr(err, "ResourceNotFoundException") && !isErr(err, "InvalidParameter") {
					return nil, err
				}
				tags = &glacier.ListTagsForVaultOutput{}
			}

			values = append(values, Resource{
				ARN:  *vault.VaultARN,
				Name: *vault.VaultName,
				Description: model.GlacierVaultDescription{
					Vault:        vault,
					AccessPolicy: *accessPolicy.Policy,
					LockPolicy: types.VaultLockPolicy{
						Policy: lockPolicy.Policy,
					},
					Tags: tags.Tags,
				},
			})
		}
	}

	return values, nil
}

func GetGlacierVault(ctx context.Context, cfg aws.Config, fields map[string]string) ([]Resource, error) {
	vaultName := fields["name"]
	describeCtx := GetDescribeContext(ctx)

	client := glacier.NewFromConfig(cfg)
	vault, err := client.DescribeVault(ctx, &glacier.DescribeVaultInput{
		AccountId: &describeCtx.AccountID,
		VaultName: &vaultName,
	})
	if err != nil {
		if !isErr(err, "ResourceNotFoundException") && !isErr(err, "InvalidParameter") {
			return nil, err
		}
		return nil, nil
	}

	var values []Resource
	accessPolicy, err := client.GetVaultAccessPolicy(ctx, &glacier.GetVaultAccessPolicyInput{
		AccountId: &describeCtx.AccountID,
		VaultName: vault.VaultName,
	})
	if err != nil {
		if !isErr(err, "ResourceNotFoundException") && !isErr(err, "InvalidParameter") {
			return nil, err
		}
		accessPolicy = &glacier.GetVaultAccessPolicyOutput{
			Policy: &types.VaultAccessPolicy{},
		}
	}

	lockPolicy, err := client.GetVaultLock(ctx, &glacier.GetVaultLockInput{
		AccountId: &describeCtx.AccountID,
		VaultName: vault.VaultName,
	})
	if err != nil {
		if !isErr(err, "ResourceNotFoundException") && !isErr(err, "InvalidParameter") {
			return nil, err
		}
		lockPolicy = &glacier.GetVaultLockOutput{}
	}

	tags, err := client.ListTagsForVault(ctx, &glacier.ListTagsForVaultInput{
		AccountId: &describeCtx.AccountID,
		VaultName: vault.VaultName,
	})
	if err != nil {
		if !isErr(err, "ResourceNotFoundException") && !isErr(err, "InvalidParameter") {
			return nil, err
		}
		tags = &glacier.ListTagsForVaultOutput{}
	}

	values = append(values, Resource{
		ARN:  *vault.VaultARN,
		Name: *vault.VaultName,
		Description: model.GlacierVaultDescription{
			Vault: types.DescribeVaultOutput{
				CreationDate:      vault.CreationDate,
				LastInventoryDate: vault.LastInventoryDate,
				NumberOfArchives:  vault.NumberOfArchives,
				SizeInBytes:       vault.SizeInBytes,
				VaultARN:          vault.VaultARN,
				VaultName:         vault.VaultName,
			},
			AccessPolicy: *accessPolicy.Policy,
			LockPolicy: types.VaultLockPolicy{
				Policy: lockPolicy.Policy,
			},
			Tags: tags.Tags,
		},
	})

	return values, nil
}
