package describe

import (
	"encoding/json"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
)

const MAX_INT64 = 9223372036854775807

var DoDescribeJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "describe_worker",
	Name:      "do_describe_jobs_total",
	Help:      "Count of done describe jobs in describe-worker service",
}, []string{"provider", "resource_type", "status"})

var DoDescribeJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "keibi",
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
	AccountID     string   `json:"accountId"`
	Regions       []string `json:"regions"`
	SecretKey     string   `json:"secretKey"`
	AccessKey     string   `json:"accessKey"`
	SessionToken  string   `json:"sessionToken"`
	AssumeRoleARN string   `json:"assumeRoleARN"`
}

func AWSAccountConfigFromMap(m map[string]interface{}) (AWSAccountConfig, error) {
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

// Do will perform the job which includes the following tasks:
//
//  1. Describing resources from the cloud providee based on the job definition.
//  2. Send the described resources to Kafka to be consumed by other systems.
//
// There are a variety of things that could go wrong in the process. This method will
// do its best to complete the task even if some errors occur along the way. However,
// if any error occurs, The JobResult will indicate that through the Status and Error
// will be set to the first error that occured.
//func (j DescribeJob) Do(ctx context.Context, vlt *vault.KMSVaultSourceConfig, keyARN string, rdb *redis.Client, logger *zap.Logger, describeDeliverEndpoint *string) {
//	logger.Info("Starting DescribeJob", zap.Uint("jobID", j.JobID), zap.Uint("scheduleJobID", j.ScheduleJobID), zap.Uint("parentJobID", j.ParentJobID), zap.String("resourceType", j.ResourceType), zap.String("sourceID", j.SourceID), zap.String("accountID", j.AccountID), zap.Int64("describedAt", j.DescribedAt), zap.String("sourceType", string(j.SourceType)), zap.String("configReg", j.CipherText), zap.String("triggerType", string(j.TriggerType)), zap.Uint("retryCounter", j.RetryCounter))
//
//	startTime := time.Now().Unix()
//	defer func() {
//		if err := recover(); err != nil {
//			fmt.Println("paniced with error:", err)
//			fmt.Println(errors.Wrap(err, 2).ErrorStack())
//		}
//	}()
//
//	// Assume it succeeded unless it fails somewhere
//	var (
//		status               = api.DescribeResourceJobSucceeded
//		firstErr    error    = nil
//		resourceIDs []string = nil
//	)
//
//	fail := func(err error) {
//		status = api.DescribeResourceJobFailed
//		if firstErr == nil {
//			firstErr = err
//		}
//	}
//
//	ctx, cancel := context.WithTimeout(ctx, describeJobTimeout)
//	defer cancel()
//
//	if conn, err := grpc.Dial(*describeDeliverEndpoint); err == nil {
//		defer conn.Close()
//		client := golang.NewDescribeServiceClient(conn)
//
//		if config, err := vlt.Decrypt(j.CipherText, keyARN); err == nil {
//			_, resourceIDs, err = doDescribe(ctx, rdb, j, config, logger, &client)
//			if err != nil {
//				// Don't return here. In certain cases, such as AWS, resources might be
//				// available for some regions while there was failures in other regions.
//				// Instead, continue to write whatever you can to kafka.
//				fail(fmt.Errorf("describe resources: %w", err))
//			}
//		} else if config == nil {
//			fail(fmt.Errorf("config is null! path is: %s", j.CipherText))
//		} else {
//			fail(fmt.Errorf("resource source config: %w", err))
//		}
//
//		errMsg := ""
//		if firstErr != nil {
//			errMsg = firstErr.Error()
//		}
//		if status == api.DescribeResourceJobSucceeded {
//			DoDescribeJobsDuration.WithLabelValues(string(j.SourceType), j.ResourceType, "successful").Observe(float64(time.Now().Unix() - startTime))
//			DoDescribeJobsCount.WithLabelValues(string(j.SourceType), j.ResourceType, "successful").Inc()
//		}
//
//		_, err := client.DeliverResult(ctx, &golang.DeliverResultRequest{
//			JobId:       uint32(j.JobID),
//			ParentJobId: uint32(j.ParentJobID),
//			Status:      string(status),
//			Error:       errMsg,
//			DescribeJob: &golang.DescribeJob{
//				JobId:         uint32(j.JobID),
//				ScheduleJobId: uint32(j.ScheduleJobID),
//				ParentJobId:   uint32(j.ParentJobID),
//				ResourceType:  j.ResourceType,
//				SourceId:      j.SourceID,
//				AccountId:     j.AccountID,
//				DescribedAt:   j.DescribedAt,
//				SourceType:    string(j.SourceType),
//				ConfigReg:     j.CipherText,
//				TriggerType:   string(j.TriggerType),
//				RetryCounter:  uint32(j.RetryCounter),
//			},
//			DescribedResourceIds: resourceIDs,
//		})
//		if err != nil {
//			fail(fmt.Errorf("DeliverResult: %v", err))
//		}
//	} else {
//		fail(fmt.Errorf("grpc: %v", err))
//	}
//}
