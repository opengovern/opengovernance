package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/streamanalytics/mgmt/2016-03-01/streamanalytics"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func StreamAnalyticsJob(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := insights.NewDiagnosticSettingsClient(subscription)
	client.Authorizer = authorizer

	streamingJobsClient := streamanalytics.NewStreamingJobsClient(subscription)
	streamingJobsClient.Authorizer = authorizer

	result, err := streamingJobsClient.List(context.Background(), "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, streamingJob := range result.Values() {
			resourceGroup := strings.Split(*streamingJob.ID, "/")[4]

			streamanalyticsListOp, err := client.List(ctx, *streamingJob.ID)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:       *streamingJob.ID,
				Location: *streamingJob.Location,
				Description: model.StreamAnalyticsJobDescription{
					StreamingJob:                streamingJob,
					DiagnosticSettingsResources: streamanalyticsListOp.Value,
					ResourceGroup:               resourceGroup,
				},
			})
		}
		if !result.NotDone() {
			break
		}
		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}
