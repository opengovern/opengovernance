package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/synthetics"
)

func SyntheticsCanary(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := synthetics.NewFromConfig(cfg)
	paginator := synthetics.NewDescribeCanariesPaginator(client, &synthetics.DescribeCanariesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Canaries {
			values = append(values, v)
		}
	}

	return values, nil
}
