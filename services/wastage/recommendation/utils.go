package recommendation

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	aws "github.com/kaytu-io/plugin-aws/plugin/proto/src/golang"
	gcp "github.com/kaytu-io/plugin-gcp/plugin/proto/src/golang/gcp"
	"google.golang.org/protobuf/types/known/wrapperspb"
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

func funcPWrapper(a, b *wrapperspb.DoubleValue, f func(aa, bb float64) float64) *wrapperspb.DoubleValue {
	if a == nil && b == nil {
		return nil
	} else if a == nil {
		return b
	} else if b == nil {
		return a
	} else {
		tmp := f(a.GetValue(), b.GetValue())
		return wrapperspb.Double(tmp)
	}
}

func getValueOrZero[T float64 | float32 | int | int64 | int32](v *T) T {
	if v == nil {
		return T(0)
	}
	return *v
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
			continue
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
			continue
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
			continue
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

func MergeGrpcDatapoints(in []*aws.Datapoint, out []*aws.Datapoint, mergeF func(aa, bb float64) float64) []*aws.Datapoint {
	dps := map[int64]*aws.Datapoint{}
	for _, dp := range in {
		dp := dp
		dps[dp.Timestamp.AsTime().Unix()] = dp
	}
	for _, dp := range out {
		dp := dp
		if dps[dp.Timestamp.AsTime().Unix()] == nil {
			dps[dp.Timestamp.AsTime().Unix()] = dp
			continue
		}

		dps[dp.Timestamp.AsTime().Unix()].Average = Float64ToWrapper(funcP(WrappedToFloat64(dps[dp.Timestamp.AsTime().Unix()].Average), WrappedToFloat64(dp.Average), mergeF))
		dps[dp.Timestamp.AsTime().Unix()].Maximum = Float64ToWrapper(funcP(WrappedToFloat64(dps[dp.Timestamp.AsTime().Unix()].Maximum), WrappedToFloat64(dp.Maximum), mergeF))
		dps[dp.Timestamp.AsTime().Unix()].Minimum = Float64ToWrapper(funcP(WrappedToFloat64(dps[dp.Timestamp.AsTime().Unix()].Minimum), WrappedToFloat64(dp.Minimum), mergeF))
		dps[dp.Timestamp.AsTime().Unix()].SampleCount = Float64ToWrapper(funcP(WrappedToFloat64(dps[dp.Timestamp.AsTime().Unix()].SampleCount), WrappedToFloat64(dp.SampleCount), mergeF))
		dps[dp.Timestamp.AsTime().Unix()].Sum = Float64ToWrapper(funcP(WrappedToFloat64(dps[dp.Timestamp.AsTime().Unix()].Sum), WrappedToFloat64(dp.Sum), mergeF))
	}

	var dpArr []*aws.Datapoint
	for _, dp := range dps {
		dpArr = append(dpArr, dp)
	}
	sort.Slice(dpArr, func(i, j int) bool {
		return dpArr[i].Timestamp.AsTime().Unix() < dpArr[j].Timestamp.AsTime().Unix()
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

func minOfAverageOfDatapoints(datapoints []types.Datapoint) *float64 {
	if len(datapoints) == 0 {
		return nil
	}

	hasNonNil := false
	minOfAverages := float64(0)
	for _, dp := range datapoints {
		dp := dp
		if dp.Average == nil {
			continue
		}
		if !hasNonNil {
			minOfAverages = *dp.Average
		}
		hasNonNil = true
		minOfAverages = min(minOfAverages, *dp.Average)
	}
	if !hasNonNil {
		return nil
	}
	return &minOfAverages
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
		minV, avgV, maxV = minOfAverageOfDatapoints(dps), averageOfDatapoints(dps), maxOfAverageOfDatapoints(dps)
	case UsageAverageTypeMax:
		minV, avgV, maxV = minOfAverageOfDatapoints(dps), maxOfAverageOfDatapoints(dps), maxOfDatapoints(dps)
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

func averageOfGCPDatapoints(datapoints []*gcp.DataPoint) *float64 {
	if len(datapoints) == 0 {
		return nil
	}

	hasNonNil := false
	avg := float64(0)
	for _, dp := range datapoints {
		hasNonNil = true
		avg += dp.Value
	}
	if !hasNonNil {
		return nil
	}
	avg = avg / float64(len(datapoints))
	return &avg
}

func maxOfGCPDatapoints(datapoints []*gcp.DataPoint) *float64 {
	if len(datapoints) == 0 {
		return nil
	}

	hasNonNil := false
	maxOfAvgs := float64(0)
	for _, dp := range datapoints {
		hasNonNil = true
		maxOfAvgs = max(maxOfAvgs, dp.Value)
	}
	if !hasNonNil {
		return nil
	}
	return &maxOfAvgs
}

func minOfGCPDatapoints(datapoints []*gcp.DataPoint) *float64 {
	if len(datapoints) == 0 {
		return nil
	}

	hasNonNil := false
	minOfAverages := float64(0)
	for _, dp := range datapoints {
		if !hasNonNil {
			minOfAverages = dp.Value
		}
		hasNonNil = true
		minOfAverages = min(minOfAverages, dp.Value)
	}
	if !hasNonNil {
		return nil
	}
	return &minOfAverages
}

func extractGCPUsage(ts []*gcp.DataPoint) gcp.Usage {
	var minV, avgV, maxV *float64
	var minW, avgW, maxW *wrapperspb.DoubleValue
	minV, avgV, maxV = minOfGCPDatapoints(ts), averageOfGCPDatapoints(ts), maxOfGCPDatapoints(ts)

	if minV != nil {
		minW = wrapperspb.Double(*minV)
	}
	if avgV != nil {
		avgW = wrapperspb.Double(*avgV)
	}
	if maxV != nil {
		maxW = wrapperspb.Double(*maxV)
	}

	return gcp.Usage{
		Avg: avgW,
		Min: minW,
		Max: maxW,
	}
}
