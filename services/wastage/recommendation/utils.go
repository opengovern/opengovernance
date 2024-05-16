package recommendation

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"math"
	"sort"
)

func funcP(a, b *float64, f func(aa, bb float64) float64) *float64 {
	if a == nil && b == nil {
		return nil
	} else if a == nil {
		return b
	} else if b == nil {
		return a
	} else {
		tmp := f(*a, *b)
		return &tmp
	}
}

func mergeDatapoints(in []types.Datapoint, out []types.Datapoint) []types.Datapoint {
	avg := func(aa, bb float64) float64 {
		return (aa + bb) / 2.0
	}
	sum := func(aa, bb float64) float64 {
		return aa + bb
	}

	dps := map[int64]*types.Datapoint{}
	for _, dp := range in {
		dp := dp
		dps[dp.Timestamp.Unix()] = &dp
	}
	for _, dp := range out {
		dp := dp
		if dps[dp.Timestamp.Unix()] == nil {
			dps[dp.Timestamp.Unix()] = &dp
			break
		}

		dps[dp.Timestamp.Unix()].Average = funcP(dps[dp.Timestamp.Unix()].Average, dp.Average, avg)
		dps[dp.Timestamp.Unix()].Maximum = funcP(dps[dp.Timestamp.Unix()].Maximum, dp.Maximum, math.Max)
		dps[dp.Timestamp.Unix()].Minimum = funcP(dps[dp.Timestamp.Unix()].Minimum, dp.Minimum, math.Min)
		dps[dp.Timestamp.Unix()].SampleCount = funcP(dps[dp.Timestamp.Unix()].SampleCount, dp.SampleCount, sum)
		dps[dp.Timestamp.Unix()].Sum = funcP(dps[dp.Timestamp.Unix()].Sum, dp.Sum, sum)
	}

	var dpArr []types.Datapoint
	for _, dp := range dps {
		dpArr = append(dpArr, *dp)
	}
	sort.Slice(dpArr, func(i, j int) bool {
		return dpArr[i].Timestamp.Unix() < dpArr[j].Timestamp.Unix()
	})
	return dpArr
}

func sumMergeDatapoints(in []types.Datapoint, out []types.Datapoint) []types.Datapoint {
	sum := func(aa, bb float64) float64 {
		return aa + bb
	}

	dps := map[int64]*types.Datapoint{}
	for _, dp := range in {
		dp := dp
		dps[dp.Timestamp.Unix()] = &dp
	}
	for _, dp := range out {
		dp := dp
		if dps[dp.Timestamp.Unix()] == nil {
			dps[dp.Timestamp.Unix()] = &dp
			break
		}

		dps[dp.Timestamp.Unix()].Average = funcP(dps[dp.Timestamp.Unix()].Average, dp.Average, sum)
		dps[dp.Timestamp.Unix()].Maximum = funcP(dps[dp.Timestamp.Unix()].Maximum, dp.Maximum, sum)
		dps[dp.Timestamp.Unix()].Minimum = funcP(dps[dp.Timestamp.Unix()].Minimum, dp.Minimum, sum)
		dps[dp.Timestamp.Unix()].SampleCount = funcP(dps[dp.Timestamp.Unix()].SampleCount, dp.SampleCount, sum)
		dps[dp.Timestamp.Unix()].Sum = funcP(dps[dp.Timestamp.Unix()].Sum, dp.Sum, sum)
	}

	var dpArr []types.Datapoint
	for _, dp := range dps {
		dpArr = append(dpArr, *dp)
	}
	sort.Slice(dpArr, func(i, j int) bool {
		return dpArr[i].Timestamp.Unix() < dpArr[j].Timestamp.Unix()
	})
	return dpArr

}

func MergeDatapoints(in []types.Datapoint, out []types.Datapoint, mergeF func(aa, bb float64) float64) []types.Datapoint {
	dps := map[int64]*types.Datapoint{}
	for _, dp := range in {
		dp := dp
		dps[dp.Timestamp.Unix()] = &dp
	}
	for _, dp := range out {
		dp := dp
		if dps[dp.Timestamp.Unix()] == nil {
			dps[dp.Timestamp.Unix()] = &dp
			break
		}

		dps[dp.Timestamp.Unix()].Average = funcP(dps[dp.Timestamp.Unix()].Average, dp.Average, mergeF)
		dps[dp.Timestamp.Unix()].Maximum = funcP(dps[dp.Timestamp.Unix()].Maximum, dp.Maximum, mergeF)
		dps[dp.Timestamp.Unix()].Minimum = funcP(dps[dp.Timestamp.Unix()].Minimum, dp.Minimum, mergeF)
		dps[dp.Timestamp.Unix()].SampleCount = funcP(dps[dp.Timestamp.Unix()].SampleCount, dp.SampleCount, mergeF)
		dps[dp.Timestamp.Unix()].Sum = funcP(dps[dp.Timestamp.Unix()].Sum, dp.Sum, mergeF)
	}

	var dpArr []types.Datapoint
	for _, dp := range dps {
		dpArr = append(dpArr, *dp)
	}
	sort.Slice(dpArr, func(i, j int) bool {
		return dpArr[i].Timestamp.Unix() < dpArr[j].Timestamp.Unix()
	})
	return dpArr

}

func averageOfDatapoints(datapoints []types.Datapoint) *float64 {
	if len(datapoints) == 0 {
		return nil
	}

	hasNonNil := false
	avg := float64(0)
	for _, dp := range datapoints {
		dp := dp
		if dp.Average == nil {
			continue
		}
		hasNonNil = true
		avg += *dp.Average
	}
	if !hasNonNil {
		return nil
	}
	avg = avg / float64(len(datapoints))
	return &avg
}

func maxOfAverageOfDatapoints(datapoints []types.Datapoint) *float64 {
	if len(datapoints) == 0 {
		return nil
	}

	hasNonNil := false
	maxOfAvgs := float64(0)
	for _, dp := range datapoints {
		dp := dp
		if dp.Average == nil {
			continue
		}
		hasNonNil = true
		maxOfAvgs = max(maxOfAvgs, *dp.Average)
	}
	if !hasNonNil {
		return nil
	}
	return &maxOfAvgs
}

func minOfDatapoints(datapoints []types.Datapoint) *float64 {
	if len(datapoints) == 0 {
		return nil
	}

	hasNonNil := false
	minV := math.MaxFloat64
	for _, dp := range datapoints {
		dp := dp
		if dp.Minimum == nil {
			continue
		}
		hasNonNil = true
		minV = min(minV, *dp.Minimum)
	}
	if !hasNonNil {
		return nil
	}
	return &minV
}

func maxOfDatapoints(datapoints []types.Datapoint) *float64 {
	if len(datapoints) == 0 {
		return nil
	}

	hasNonNil := false
	maxV := 0.0
	for _, dp := range datapoints {
		dp := dp
		if dp.Maximum == nil {
			continue
		}
		hasNonNil = true
		maxV = max(maxV, *dp.Maximum)
	}
	if !hasNonNil {
		return nil
	}
	return &maxV
}

type UsageAverageType int

const (
	UsageAverageTypeAverage UsageAverageType = iota
	UsageAverageTypeMax
)

func extractUsage(dps []types.Datapoint, avgType UsageAverageType) entity.Usage {
	var minV, avgV, maxV *float64
	switch avgType {
	case UsageAverageTypeAverage:
		minV, avgV, maxV = minOfDatapoints(dps), averageOfDatapoints(dps), maxOfAverageOfDatapoints(dps)
	case UsageAverageTypeMax:
		minV, avgV, maxV = minOfDatapoints(dps), maxOfAverageOfDatapoints(dps), maxOfDatapoints(dps)
	}

	var lastDP *types.Datapoint
	if len(dps) > 0 {
		lastDP = &dps[len(dps)-1]
	}

	return entity.Usage{
		Avg:  avgV,
		Min:  minV,
		Max:  maxV,
		Last: lastDP,
	}
}
