package describer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

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
			// This prevents Implicit memory aliasing in for loop
			detectorId := detectorId

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
						ARN:  *item.Arn,
						Name: *item.Id,
						Description: model.GuardDutyFindingDescription{
							Finding: item,
						},
					})
				}
			}
		}
	}
	return values, nil
}

func GuardDutyDetector(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	var values []Resource

	client := guardduty.NewFromConfig(cfg)

	describeCtx := GetDescribeContext(ctx)

	paginator := guardduty.NewListDetectorsPaginator(client, &guardduty.ListDetectorsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, id := range page.DetectorIds {
			id := id
			out, err := client.GetDetector(ctx, &guardduty.GetDetectorInput{
				DetectorId: &id,
			})
			if err != nil {
				return nil, err
			}

			arn := "arn:" + describeCtx.Partition + ":guardduty:" + describeCtx.Region + ":" + describeCtx.AccountID + ":detector/" + id
			values = append(values, Resource{
				ARN:  arn,
				Name: id,
				Description: model.GuardDutyDetectorDescription{
					DetectorId: id,
					Detector:   out,
				},
			})
		}
	}
	return values, nil
}

func GuardDutyFilter(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := guardduty.NewFromConfig(cfg)
	paginator := guardduty.NewListDetectorsPaginator(client, &guardduty.ListDetectorsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, detectorId := range page.DetectorIds {
			filterPaginator := guardduty.NewListFiltersPaginator(client, &guardduty.ListFiltersInput{
				DetectorId: &detectorId,
			})

			for filterPaginator.HasMorePages() {
				filterPage, err := filterPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for _, filter := range filterPage.FilterNames {
					arn := fmt.Sprintf("arn:%s:guardduty:%s:%s:detector/%s/filter/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, detectorId, filter)

					filterOutput, err := client.GetFilter(ctx, &guardduty.GetFilterInput{
						DetectorId: &detectorId,
						FilterName: &filter,
					})
					if err != nil {
						return nil, err
					}

					values = append(values, Resource{
						ARN:  arn,
						Name: *filterOutput.Name,
						Description: model.GuardDutyFilterDescription{
							Filter:     *filterOutput,
							DetectorId: detectorId,
						},
					})
				}
			}
		}
	}
	return values, nil
}

func GuardDutyIPSet(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := guardduty.NewFromConfig(cfg)
	paginator := guardduty.NewListDetectorsPaginator(client, &guardduty.ListDetectorsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, detectorId := range page.DetectorIds {
			ipSetsPaginator := guardduty.NewListIPSetsPaginator(client, &guardduty.ListIPSetsInput{
				DetectorId: &detectorId,
			})

			for ipSetsPaginator.HasMorePages() {
				ipSetPage, err := ipSetsPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for _, ipSetId := range ipSetPage.IpSetIds {
					arn := fmt.Sprintf("arn:%s:guardduty:%s:%s:detector/%s/ipset/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, detectorId, ipSetId)

					ipSetOutput, err := client.GetIPSet(ctx, &guardduty.GetIPSetInput{
						DetectorId: &detectorId,
						IpSetId:    &ipSetId,
					})
					if err != nil {
						return nil, err
					}

					values = append(values, Resource{
						ARN:  arn,
						Name: *ipSetOutput.Name,
						Description: model.GuardDutyIPSetDescription{
							IPSet:      *ipSetOutput,
							DetectorId: detectorId,
							IPSetId:    ipSetId,
						},
					})
				}
			}
		}
	}
	return values, nil
}

func GuardDutyMember(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	//describeCtx := GetDescribeContext(ctx)
	client := guardduty.NewFromConfig(cfg)
	paginator := guardduty.NewListDetectorsPaginator(client, &guardduty.ListDetectorsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, detectorId := range page.DetectorIds {
			membersPaginator := guardduty.NewListMembersPaginator(client, &guardduty.ListMembersInput{
				DetectorId: &detectorId,
			})

			for membersPaginator.HasMorePages() {
				membersPage, err := membersPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for _, member := range membersPage.Members {
					values = append(values, Resource{
						Name: *member.AccountId,
						Description: model.GuardDutyMemberDescription{
							Member: member,
						},
					})
				}
			}
		}
	}
	return values, nil
}

func GuardDutyPublishingDestination(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := guardduty.NewFromConfig(cfg)
	paginator := guardduty.NewListDetectorsPaginator(client, &guardduty.ListDetectorsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, detectorId := range page.DetectorIds {
			publishingDestinationsPaginator := guardduty.NewListPublishingDestinationsPaginator(client, &guardduty.ListPublishingDestinationsInput{
				DetectorId: &detectorId,
			})

			for publishingDestinationsPaginator.HasMorePages() {
				publishingDestinationsPage, err := publishingDestinationsPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for _, destination := range publishingDestinationsPage.Destinations {
					arn := fmt.Sprintf("arn:%s:guardduty:%s:%s:detector/%s/publishingDestination/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, detectorId, *destination.DestinationId)

					destinationOutput, err := client.DescribePublishingDestination(ctx, &guardduty.DescribePublishingDestinationInput{
						DestinationId: destination.DestinationId,
						DetectorId:    &detectorId,
					})
					if err != nil {
						return nil, err
					}

					values = append(values, Resource{
						ARN: arn,
						ID:  *destinationOutput.DestinationId,
						Description: model.GuardDutyPublishingDestinationDescription{
							PublishingDestination: *destinationOutput,
							DetectorId:            detectorId,
						},
					})
				}
			}
		}
	}
	return values, nil
}

func GuardDutyThreatIntelSet(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := guardduty.NewFromConfig(cfg)
	paginator := guardduty.NewListDetectorsPaginator(client, &guardduty.ListDetectorsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, detectorId := range page.DetectorIds {
			threatIntelSetsPaginator := guardduty.NewListThreatIntelSetsPaginator(client, &guardduty.ListThreatIntelSetsInput{
				DetectorId: &detectorId,
			})

			for threatIntelSetsPaginator.HasMorePages() {
				threatIntelSetsPage, err := threatIntelSetsPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for _, threatIntelSetId := range threatIntelSetsPage.ThreatIntelSetIds {
					arn := fmt.Sprintf("arn:%s:guardduty:%s:%s:detector/%s/threatintelset/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, detectorId, threatIntelSetId)

					threatIntelSetOutput, err := client.GetThreatIntelSet(ctx, &guardduty.GetThreatIntelSetInput{
						DetectorId:       &detectorId,
						ThreatIntelSetId: &threatIntelSetId,
					})
					if err != nil {
						return nil, err
					}

					values = append(values, Resource{
						ARN:  arn,
						Name: *threatIntelSetOutput.Name,
						Description: model.GuardDutyThreatIntelSetDescription{
							ThreatIntelSet:   *threatIntelSetOutput,
							DetectorId:       detectorId,
							ThreatIntelSetID: threatIntelSetId,
						},
					})
				}
			}
		}
	}
	return values, nil
}
