package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func KinesisStream(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := kinesis.NewFromConfig(cfg)

	var values []Resource
	var lastStreamName *string = nil
	for {
		streams, err := client.ListStreams(ctx, &kinesis.ListStreamsInput{
			ExclusiveStartStreamName: lastStreamName,
		})
		if err != nil {
			if isErr(err, "ResourceNotFoundException") || isErr(err, "InvalidParameter") {
				return nil, nil
			}
			return nil, err
		}
		for _, streamName := range streams.StreamNames {
			streamName := streamName
			stream, err := client.DescribeStream(ctx, &kinesis.DescribeStreamInput{
				StreamName: &streamName,
			})
			if err != nil {
				if !isErr(err, "ResourceNotFoundException") && !isErr(err, "InvalidParameter") {
					return nil, err
				}
				continue
			}

			streamSummery, err := client.DescribeStreamSummary(ctx, &kinesis.DescribeStreamSummaryInput{
				StreamName: &streamName,
			})
			if err != nil {
				if !isErr(err, "ResourceNotFoundException") && !isErr(err, "InvalidParameter") {
					return nil, err
				}
				continue
			}

			tags, err := client.ListTagsForStream(ctx, &kinesis.ListTagsForStreamInput{
				StreamName: &streamName,
			})
			if err != nil {
				if !isErr(err, "ResourceNotFoundException") && !isErr(err, "InvalidParameter") {
					return nil, err
				}
				tags = &kinesis.ListTagsForStreamOutput{}
			}

			values = append(values, Resource{
				ARN:  *stream.StreamDescription.StreamARN,
				Name: *stream.StreamDescription.StreamName,
				Description: model.KinesisStreamDescription{
					Stream:             *stream.StreamDescription,
					DescriptionSummary: *streamSummery.StreamDescriptionSummary,
					Tags:               tags.Tags,
				},
			})
		}

		if streams.HasMoreStreams == nil || !*streams.HasMoreStreams {
			break
		}

		lastStreamName = &streams.StreamNames[len(streams.StreamNames)-1]
	}

	return values, nil
}
