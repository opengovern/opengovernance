package describe

import (
	"time"

	"github.com/opengovern/og-util/pkg/integration"

	"github.com/opengovern/og-util/pkg/describe/enums"
	"github.com/opengovern/opengovernance/services/describe/api"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const MAX_INT64 = 9223372036854775807

var DoDescribeJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "opengovernance",
	Subsystem: "describe_worker",
	Name:      "do_describe_jobs_total",
	Help:      "Count of done describe jobs in describe-worker service",
}, []string{"provider", "resource_type", "status"})

var DoDescribeJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "opengovernance",
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
	JobID           uint // DescribeResourceJob ID
	ScheduleJobID   uint
	ParentJobID     uint // DescribeSourceJob ID
	ResourceType    string
	IntegrationID   string
	ProviderID      string
	DescribedAt     int64
	IntegrationType integration.Type
	CipherText      string
	TriggerType     enums.DescribeTriggerType
	RetryCounter    uint
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
