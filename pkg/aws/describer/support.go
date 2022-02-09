package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/support"
	"github.com/aws/aws-sdk-go-v2/service/support/types"
)

// DescribeServicesByLang Describes the specified services running in your
// cluster with a specified language. e.g. "EN"
func DescribeServicesByLang(cfg aws.Config, lang string) ([]types.Service, error) {
	svc := support.NewFromConfig(cfg)

	req, err := svc.DescribeServices(context.Background(), &support.DescribeServicesInput{Language: aws.String(lang)})
	if err != nil {
		return nil, err
	}

	return req.Services, nil
}
