package recommendation

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	types2 "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMaxNull(t *testing.T) {
	volumeMetrics := map[string]map[string][]types2.Datapoint{}

	n := time.Now()
	volumeMetrics["vol1"] = map[string][]types2.Datapoint{
		"VolumeReadOps": []types2.Datapoint{
			{
				Average:            aws.Float64(0),
				ExtendedStatistics: nil,
				Maximum:            nil,
				Minimum:            nil,
				SampleCount:        nil,
				Sum:                nil,
				Timestamp:          aws.Time(n),
				Unit:               "",
			},
			{
				Average:            aws.Float64(0),
				ExtendedStatistics: nil,
				Maximum:            nil,
				Minimum:            nil,
				SampleCount:        nil,
				Sum:                nil,
				Timestamp:          aws.Time(n.Add(1 * time.Hour)),
				Unit:               "",
			},
			{
				Average:            aws.Float64(1),
				ExtendedStatistics: nil,
				Maximum:            nil,
				Minimum:            nil,
				SampleCount:        nil,
				Sum:                nil,
				Timestamp:          aws.Time(n.Add(2 * time.Hour)),
				Unit:               "",
			},
		},
		"VolumeWriteOps": []types2.Datapoint{
			{
				Average:            aws.Float64(2),
				ExtendedStatistics: nil,
				Maximum:            nil,
				Minimum:            nil,
				SampleCount:        nil,
				Sum:                nil,
				Timestamp:          aws.Time(n),
				Unit:               "",
			},
			{
				Average:            aws.Float64(2),
				ExtendedStatistics: nil,
				Maximum:            nil,
				Minimum:            nil,
				SampleCount:        nil,
				Sum:                nil,
				Timestamp:          aws.Time(n.Add(1 * time.Hour)),
				Unit:               "",
			},
			{
				Average:            aws.Float64(3),
				ExtendedStatistics: nil,
				Maximum:            nil,
				Minimum:            nil,
				SampleCount:        nil,
				Sum:                nil,
				Timestamp:          aws.Time(n.Add(2 * time.Hour)),
				Unit:               "",
			},
		},
	}

	var ebsIopsDatapoints []types2.Datapoint
	for _, v := range volumeMetrics {
		ebsIopsDatapoints = mergeDatapoints(sumMergeDatapoints(v["VolumeReadOps"], v["VolumeWriteOps"]), ebsIopsDatapoints)
	}

	usage := extractUsage(ebsIopsDatapoints, UsageAverageTypeMax)
	assert.Nil(t, usage.Max)
	assert.Equal(t, 4.0, *usage.Avg)
}

func TestMaxNotNull(t *testing.T) {
	volumeMetrics := map[string]map[string][]types2.Datapoint{}

	n := time.Now()
	volumeMetrics["vol1"] = map[string][]types2.Datapoint{
		"VolumeReadOps": []types2.Datapoint{
			{
				Average:            aws.Float64(0),
				ExtendedStatistics: nil,
				Maximum:            aws.Float64(1),
				Minimum:            nil,
				SampleCount:        nil,
				Sum:                nil,
				Timestamp:          aws.Time(n),
				Unit:               "",
			},
			{
				Average:            aws.Float64(0),
				ExtendedStatistics: nil,
				Maximum:            nil,
				Minimum:            nil,
				SampleCount:        nil,
				Sum:                nil,
				Timestamp:          aws.Time(n.Add(1 * time.Hour)),
				Unit:               "",
			},
			{
				Average:            aws.Float64(1),
				ExtendedStatistics: nil,
				Maximum:            nil,
				Minimum:            nil,
				SampleCount:        nil,
				Sum:                nil,
				Timestamp:          aws.Time(n.Add(2 * time.Hour)),
				Unit:               "",
			},
		},
		"VolumeWriteOps": []types2.Datapoint{
			{
				Average:            aws.Float64(2),
				ExtendedStatistics: nil,
				Maximum:            nil,
				Minimum:            nil,
				SampleCount:        nil,
				Sum:                nil,
				Timestamp:          aws.Time(n),
				Unit:               "",
			},
			{
				Average:            aws.Float64(2),
				ExtendedStatistics: nil,
				Maximum:            nil,
				Minimum:            nil,
				SampleCount:        nil,
				Sum:                nil,
				Timestamp:          aws.Time(n.Add(1 * time.Hour)),
				Unit:               "",
			},
			{
				Average:            aws.Float64(3),
				ExtendedStatistics: nil,
				Maximum:            nil,
				Minimum:            nil,
				SampleCount:        nil,
				Sum:                nil,
				Timestamp:          aws.Time(n.Add(2 * time.Hour)),
				Unit:               "",
			},
		},
	}

	var ebsIopsDatapoints []types2.Datapoint
	for _, v := range volumeMetrics {
		ebsIopsDatapoints = mergeDatapoints(sumMergeDatapoints(v["VolumeReadOps"], v["VolumeWriteOps"]), ebsIopsDatapoints)
	}

	usage := extractUsage(ebsIopsDatapoints, UsageAverageTypeMax)
	assert.Equal(t, 1.0, *usage.Max)
	assert.Equal(t, 4.0, *usage.Avg)
}
