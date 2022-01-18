package describer

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/guardduty/types"
)

type GuardDutyFindingDescription struct {
	Finding types.Finding
}

func GuardDutyFinding(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	var values []Resource

	client := guardduty.NewFromConfig(cfg)

	dpaginator := guardduty.NewListDetectorsPaginator(client, &guardduty.ListDetectorsInput{})
	for dpaginator.HasMorePages() {
		dpage, err := dpaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, detectorId := range dpage.DetectorIds {
			paginator := guardduty.NewListFindingsPaginator(client, &guardduty.ListFindingsInput{
				DetectorId: &detectorId,
			})

			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}

				findings, err := client.GetFindings(ctx, &guardduty.GetFindingsInput{
					DetectorId: &detectorId,
					FindingIds: page.FindingIds,
				})
				if err != nil {
					return nil, err
				}

				for _, item := range findings.Findings {
					values = append(values, Resource{
						ARN: *item.Arn,
						Description: GuardDutyFindingDescription{
							Finding: item,
						},
					})
				}
			}
		}
	}
	return values, nil
}

type GuardDutyDetectorDescription struct {
	DetectorId string
	Detector *guardduty.GetDetectorOutput
}

func GuardDutyDetector(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	var values []Resource

	client := guardduty.NewFromConfig(cfg)

	paginator := guardduty.NewListDetectorsPaginator(client, &guardduty.ListDetectorsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.DetectorIds {
			out, err := client.GetDetector(ctx, &guardduty.GetDetectorInput{
				DetectorId: &item,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID: item,
				Description: GuardDutyDetectorDescription{
					DetectorId: item,
					Detector: out,
				},
			})
		}
	}
	return values, nil
}
