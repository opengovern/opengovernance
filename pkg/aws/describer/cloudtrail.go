package describer

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type CloudTrailTrailDescription struct {
	Trail                  types.Trail
	TrailStatus            cloudtrail.GetTrailStatusOutput
	EventSelectors         []types.EventSelector
	AdvancedEventSelectors []types.AdvancedEventSelector
	Tags                   []types.Tag
}

func CloudTrailTrail(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudtrail.NewFromConfig(cfg)
	paginator := cloudtrail.NewListTrailsPaginator(client, &cloudtrail.ListTrailsInput{})

	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		var trails []string
		for _, trail := range page.Trails {
			// Ignore trails that don't belong this region (Based on steampipe)
			if !strings.EqualFold(*trail.HomeRegion, cfg.Region) {
				continue
			}

			if trail.TrailARN != nil {
				// Ignore trails that don't belong to this account (Based on steampipe)
				if aws.ToString(identity.Account) != arnToAccountId(*trail.TrailARN) {
					continue
				}

				trails = append(trails, *trail.TrailARN)
			} else if trail.Name != nil {
				trails = append(trails, *trail.Name)
			} else {
				continue
			}
		}

		output, err := client.DescribeTrails(ctx, &cloudtrail.DescribeTrailsInput{
			IncludeShadowTrails: aws.Bool(false),
			TrailNameList:       trails,
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.TrailList {
			statusOutput, err := client.GetTrailStatus(ctx, &cloudtrail.GetTrailStatusInput{
				Name: v.TrailARN,
			})
			if err != nil {
				return nil, err
			}

			selectorOutput, err := client.GetEventSelectors(ctx, &cloudtrail.GetEventSelectorsInput{
				TrailName: v.TrailARN,
			})
			if err != nil {
				return nil, err
			}

			tagsOutput, err := client.ListTags(ctx, &cloudtrail.ListTagsInput{
				ResourceIdList: []string{*v.TrailARN},
			})
			if err != nil {
				return nil, err
			}
			var tags []types.Tag
			if len(tagsOutput.ResourceTagList) > 0 {
				tags = tagsOutput.ResourceTagList[0].TagsList
			}

			values = append(values, Resource{
				ARN:  *v.TrailARN,
				Name: *v.Name,
				Description: CloudTrailTrailDescription{
					Trail:                  v,
					TrailStatus:            *statusOutput,
					EventSelectors:         selectorOutput.EventSelectors,
					AdvancedEventSelectors: selectorOutput.AdvancedEventSelectors,
					Tags:                   tags,
				},
			})
		}
	}

	return values, nil
}

func arnToAccountId(arn string) string {
	if arn != "" {
		return strings.Split(arn, ":")[4]
	}

	return ""
}
