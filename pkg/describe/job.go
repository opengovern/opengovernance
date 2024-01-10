package describe

import (
	"encoding/json"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/enums"
	"github.com/kaytu-io/kaytu-util/pkg/source"
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

type AWSAccountConfig struct {
	AccountID            string   `json:"accountId"`
	Regions              []string `json:"regions"`
	SecretKey            string   `json:"secretKey"`
	AccessKey            string   `json:"accessKey"`
	SessionToken         string   `json:"sessionToken"`
	AssumeRoleName       string   `json:"assumeRoleName"`
	AssumeAdminRoleName  string   `json:"assumeAdminRoleName"`
	AssumeRolePolicyName string   `json:"assumeRolePolicyName"`
	ExternalID           *string  `json:"externalID,omitempty"`
}

func (asc AWSAccountConfig) ToMap() map[string]any {
	jsonCnf, err := json.Marshal(asc)
	if err != nil {
		return nil
	}
	res := make(map[string]any)
	err = json.Unmarshal(jsonCnf, &res)
	if err != nil {
		return nil
	}
	return res
}

func AWSAccountConfigFromMap(m map[string]any) (AWSAccountConfig, error) {
	mj, err := json.Marshal(m)
	if err != nil {
		return AWSAccountConfig{}, err
	}

	var c AWSAccountConfig
	err = json.Unmarshal(mj, &c)
	if err != nil {
		return AWSAccountConfig{}, err
	}

	return c, nil
}

type AzureSubscriptionConfig struct {
	SubscriptionID  string `json:"subscriptionId"`
	TenantID        string `json:"tenantId"`
	ObjectID        string `json:"objectId"`
	SecretID        string `json:"secretId"`
	ClientID        string `json:"clientId"`
	ClientSecret    string `json:"clientSecret"`
	CertificatePath string `json:"certificatePath"`
	CertificatePass string `json:"certificatePass"`
	Username        string `json:"username"`
	Password        string `json:"password"`
}

func (asc AzureSubscriptionConfig) ToMap() map[string]any {
	jsonCnf, err := json.Marshal(asc)
	if err != nil {
		return nil
	}
	res := make(map[string]any)
	err = json.Unmarshal(jsonCnf, &res)
	if err != nil {
		return nil
	}
	return res
}

func AzureSubscriptionConfigFromMap(m map[string]interface{}) (AzureSubscriptionConfig, error) {
	mj, err := json.Marshal(m)
	if err != nil {
		return AzureSubscriptionConfig{}, err
	}

	var c AzureSubscriptionConfig
	err = json.Unmarshal(mj, &c)
	if err != nil {
		return AzureSubscriptionConfig{}, err
	}

	return c, nil
}

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
