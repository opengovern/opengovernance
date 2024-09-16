package describe

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/describe/enums"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/open-governance/pkg/describe/api"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const MAX_INT64 = 9223372036854775807

var DoDescribeJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "kaytu",
	Subsystem: "describe_worker",
	Name:      "do_describe_jobs_total",
	Help:      "Count of done describe jobs in describe-worker service",
}, []string{"provider", "resource_type", "status"})

var DoDescribeJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "kaytu",
	Subsystem: "describe_worker",
	Name:      "do_describe_jobs_duration_seconds",
	Help:      "Duration of done describe jobs in describe-worker service",
	Buckets:   []float64{5, 60, 300, 600, 1800, 3600, 7200, 36000},
}, []string{"provider", "resource_type", "status"})

const (
	InventorySummaryIndex = "inventory_summary"
	describeJobTimeout    = 3 * 60 * time.Minute
)

type DescribeJob struct {
	JobID         uint // DescribeResourceJob ID
	ScheduleJobID uint
	ParentJobID   uint // DescribeSourceJob ID
	ResourceType  string
	SourceID      string
	AccountID     string
	DescribedAt   int64
	SourceType    source.Type
	CipherText    string
	TriggerType   enums.DescribeTriggerType
	RetryCounter  uint
}

type DescribeJobResult struct {
	JobID                uint
	ParentJobID          uint
	Status               api.DescribeResourceJobStatus
	Error                string
	DescribeJob          DescribeJob
	DescribedResourceIDs []string
	ErrorCode            string
}

func (r DescribeJobResult) KeysAndIndex() ([]string, string) {
	return []string{}, ""
}
