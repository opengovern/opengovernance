package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer/types"
)

type AccessAnalyzerAnalyzerDescription struct {
	Analyzer types.AnalyzerSummary
	Findings []types.FindingSummary
}

func AccessAnalyzerAnalyzer(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := accessanalyzer.NewFromConfig(cfg)
	paginator := accessanalyzer.NewListAnalyzersPaginator(client, &accessanalyzer.ListAnalyzersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Analyzers {
			findings, err := getAnalyzerFindings(ctx, client, v.Arn)
			if err != nil {
				return nil, err
			}
			values = append(values, Resource{
				ARN:  *v.Arn,
				Name: *v.Name,
				Description: AccessAnalyzerAnalyzerDescription{
					Analyzer: v,
					Findings: findings,
				},
			})
		}
	}

	return values, nil
}

func getAnalyzerFindings(ctx context.Context, client *accessanalyzer.Client, analyzerArn *string) ([]types.FindingSummary, error) {
	paginator := accessanalyzer.NewListFindingsPaginator(client, &accessanalyzer.ListFindingsInput{
		AnalyzerArn: analyzerArn,
	})

	var findings []types.FindingSummary
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		findings = append(findings, page.Findings...)
	}

	return findings, nil
}
