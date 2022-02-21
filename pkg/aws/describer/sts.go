package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func GetAccountId(ctx context.Context, cfg aws.Config) (string, error) {
	svc := sts.NewFromConfig(cfg)

	acc, err := svc.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}

	return *acc.Account, nil
}
