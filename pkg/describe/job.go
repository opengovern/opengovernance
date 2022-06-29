package describe

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/go-errors/errors"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
)

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

var DoDescribeCleanupJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "describe_cleanup_worker",
	Name:      "do_describe_cleanup_jobs_total",
	Help:      "Count of done describe cleanup jobs in describe-worker service",
}, []string{"resource_type", "status"})

var DoDescribeCleanupJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "keibi",
	Subsystem: "describe_cleanup_worker",
	Name:      "do_describe_cleanup_jobs_duration_seconds",
	Help:      "Duration of done describe cleanup jobs in describe-worker service",
	Buckets:   []float64{5, 60, 300, 600, 1800, 3600, 7200, 36000},
}, []string{"resource_type", "status"})

const (
	InventorySummaryIndex  = "inventory_summary"
	SourceResourcesSummary = "source_resources_summary"
	describeJobTimeout     = 5 * time.Minute
	cleanupJobTimeout      = 5 * time.Minute
)

var stopWordsRe = regexp.MustCompile(`\W+`)

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
	JobID        uint // DescribeResourceJob ID
	ParentJobID  uint // DescribeSourceJob ID
	ResourceType string
	SourceID     string
	AccountID    string
	DescribedAt  int64
	SourceType   api.SourceType
	ConfigReg    string

	LastDaySourceJobID     uint
	LastWeekSourceJobID    uint
	LastQuarterSourceJobID uint
	LastYearSourceJobID    uint
}

type DescribeJobResult struct {
	JobID       uint
	ParentJobID uint
	Status      api.DescribeResourceJobStatus
	Error       string
}

// Do will perform the job which includes the following tasks:
//
//    1. Describing resources from the cloud providee based on the job definition.
//    2. Send the described resources to Kafka to be consumed by other systems.
//
// There are a variety of things that could go wrong in the process. This method will
// do its best to complete the task even if some errors occur along the way. However,
// if any error occurs, The JobResult will indicate that through the Status and Error
// will be set to the first error that occured.
func (j DescribeJob) Do(vlt vault.SourceConfig, es keibi.Client, producer sarama.SyncProducer, topic string, logger *zap.Logger) (r DescribeJobResult) {
	startTime := time.Now().Unix()
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("paniced with error:", err)
			fmt.Println(errors.Wrap(err, 2).ErrorStack())

			DoDescribeJobsDuration.WithLabelValues(string(j.SourceType), j.ResourceType, "failure").Observe(float64(time.Now().Unix() - startTime))
			DoDescribeJobsCount.WithLabelValues(string(j.SourceType), j.ResourceType, "failure").Inc()
			r = DescribeJobResult{
				JobID:       j.JobID,
				ParentJobID: j.ParentJobID,
				Status:      api.DescribeResourceJobFailed,
				Error:       fmt.Sprintf("paniced: %s", err),
			}
		}
	}()

	// Assume it succeeded unless it fails somewhere
	var (
		status         = api.DescribeResourceJobSucceeded
		firstErr error = nil
	)

	fail := func(err error) {
		DoDescribeJobsDuration.WithLabelValues(string(j.SourceType), j.ResourceType, "failure").Observe(float64(time.Now().Unix() - startTime))
		DoDescribeJobsCount.WithLabelValues(string(j.SourceType), j.ResourceType, "failure").Inc()
		status = api.DescribeResourceJobFailed
		if firstErr == nil {
			firstErr = err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), describeJobTimeout)
	defer cancel()

	config, err := vlt.Read(j.ConfigReg)
	if err != nil {
		fail(fmt.Errorf("resource source config: %w", err))
	} else {
		msgs, err := doDescribe(ctx, es, j, config, logger)
		if err != nil {
			// Don't return here. In certain cases, such as AWS, resources might be
			// available for some regions while there was failures in other regions.
			// Instead, continue to write whatever you can to kafka.
			fail(fmt.Errorf("describe resources: %w", err))
		}

		if err := kafka.DoSendToKafka(producer, topic, msgs, logger); err != nil {
			fail(fmt.Errorf("send to kafka: %w", err))
		}
	}

	errMsg := ""
	if firstErr != nil {
		errMsg = firstErr.Error()
	}
	if status == api.DescribeResourceJobSucceeded {
		DoDescribeJobsDuration.WithLabelValues(string(j.SourceType), j.ResourceType, "successful").Observe(float64(time.Now().Unix() - startTime))
		DoDescribeJobsCount.WithLabelValues(string(j.SourceType), j.ResourceType, "successful").Inc()
	}

	return DescribeJobResult{
		JobID:       j.JobID,
		ParentJobID: j.ParentJobID,
		Status:      status,
		Error:       errMsg,
	}
}

// doDescribe describes the sources, e.g. AWS, Azure and returns the responses.
func doDescribe(ctx context.Context, es keibi.Client, job DescribeJob, config map[string]interface{}, logger *zap.Logger) ([]kafka.DescribedResource, error) {
	logger.Info(fmt.Sprintf("Proccessing Job: ID[%d] ParentJobID[%d] RosourceType[%s]\n", job.JobID, job.ParentJobID, job.ResourceType))

	switch job.SourceType {
	case api.SourceCloudAWS:
		return doDescribeAWS(ctx, es, job, config)
	case api.SourceCloudAzure:
		return doDescribeAzure(ctx, es, job, config)
	default:
		return nil, fmt.Errorf("invalid SourceType: %s", job.SourceType)
	}
}

func doDescribeAWS(ctx context.Context, es keibi.Client, job DescribeJob, config map[string]interface{}) ([]kafka.DescribedResource, error) {
	creds, err := AWSAccountConfigFromMap(config)
	if err != nil {
		return nil, fmt.Errorf("aws account credentials: %w", err)
	}

	output, err := aws.GetResources(
		ctx,
		job.ResourceType,
		creds.AccountID,
		creds.Regions,
		creds.AccessKey,
		creds.SecretKey,
		creds.SessionToken,
		creds.AssumeRoleARN,
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("AWS: %w", err)
	}

	var errs []string
	for region, err := range output.Errors {
		if err != "" {
			errs = append(errs, fmt.Sprintf("region (%s): %s", region, err))
		}
	}

	var msgs []kafka.DescribedResource
	locationDistribution := map[string]int{}

	categoryCount := map[string]int{}
	serviceCount := map[string]int{}
	serviceDistribution := map[string]map[string]int{}
	for _, resources := range output.Resources {
		for _, resource := range resources {
			if resource.Description == nil {
				continue
			}

			msgs = append(msgs, kafka.Resource{
				ID:            resource.UniqueID(),
				Description:   resource.Description,
				SourceType:    api.SourceCloudAWS,
				ResourceType:  job.ResourceType,
				ResourceJobID: job.JobID,
				SourceJobID:   job.ParentJobID,
				SourceID:      job.SourceID,
				Metadata: map[string]string{
					"name":          resource.Name,
					"partition":     resource.Partition,
					"region":        resource.Region,
					"account_id":    resource.Account,
					"resource_type": resource.Type,
				},
			})

			msgs = append(msgs, kafka.LookupResource{
				ResourceID:    resource.UniqueID(),
				Name:          resource.Name,
				SourceType:    api.SourceCloudAWS,
				ResourceType:  job.ResourceType,
				ResourceGroup: "",
				Location:      resource.Region,
				SourceID:      job.SourceID,
				ResourceJobID: job.JobID,
				SourceJobID:   job.ParentJobID,
				CreatedAt:     job.DescribedAt,
				IsCommon:      cloudservice.IsCommonByResourceType(job.ResourceType),
			})
			region := strings.TrimSpace(resource.Region)
			if region != "" {
				locationDistribution[region]++
				if s := cloudservice.ServiceNameByResourceType(resource.Type); s != "" {
					if serviceDistribution[s] == nil {
						serviceDistribution[s] = make(map[string]int)
					}

					serviceDistribution[s][region]++
				}
			}

			if s := cloudservice.ServiceNameByResourceType(resource.Type); s != "" {
				serviceCount[s]++
			}
			if s := cloudservice.CategoryByResourceType(resource.Type); s != "" {
				if cloudservice.IsCommonByResourceType(resource.Type) {
					categoryCount[s]++
				}
			}
		}
	}

	for name, count := range serviceCount {
		var lastDayValue, lastWeekValue, lastQuarterValue, lastYearValue int
		for idx, jobID := range []uint{job.LastDaySourceJobID, job.LastWeekSourceJobID, job.LastQuarterSourceJobID,
			job.LastYearSourceJobID} {
			var response ServiceQueryResponse
			query, err := FindOldServiceValue(jobID, name)
			if err != nil {
				errs = append(errs, fmt.Sprintf("failed to build query: %v", err.Error()))
				continue
			}
			err = es.Search(context.Background(), kafka.SourceResourcesSummaryIndex, query, &response)
			if err != nil {
				errs = append(errs, fmt.Sprintf("failed to run query: %v", err.Error()))
				continue
			}

			if len(response.Hits.Hits) > 0 {
				// there will be only one result anyway
				switch idx {
				case 0:
					lastDayValue = response.Hits.Hits[0].Source.ResourceCount
				case 1:
					lastWeekValue = response.Hits.Hits[0].Source.ResourceCount
				case 2:
					lastQuarterValue = response.Hits.Hits[0].Source.ResourceCount
				case 3:
					lastYearValue = response.Hits.Hits[0].Source.ResourceCount
				}
			}
		}

		msgs = append(msgs, kafka.SourceServicesSummary{
			ServiceName:      name,
			SourceType:       job.SourceType,
			SourceJobID:      job.JobID,
			DescribedAt:      job.DescribedAt,
			ResourceCount:    count,
			LastDayCount:     lastDayValue,
			LastWeekCount:    lastWeekValue,
			LastQuarterCount: lastQuarterValue,
			LastYearCount:    lastYearValue,
			ReportType:       kafka.ResourceSummaryTypeLastServiceSummary,
		})

		msgs = append(msgs, kafka.SourceServicesSummary{
			ServiceName:      name,
			SourceType:       job.SourceType,
			SourceJobID:      job.JobID,
			DescribedAt:      job.DescribedAt,
			ResourceCount:    count,
			LastDayCount:     lastDayValue,
			LastWeekCount:    lastWeekValue,
			LastQuarterCount: lastQuarterValue,
			LastYearCount:    lastYearValue,
			ReportType:       kafka.ResourceSummaryTypeServiceHistorySummary,
		})
	}

	for name, count := range categoryCount {
		var lastDayValue, lastWeekValue, lastQuarterValue, lastYearValue int
		for idx, jobID := range []uint{job.LastDaySourceJobID, job.LastWeekSourceJobID, job.LastQuarterSourceJobID,
			job.LastYearSourceJobID} {
			var response CategoryQueryResponse
			query, err := FindOldCategoryValue(jobID, name)
			if err != nil {
				errs = append(errs, fmt.Sprintf("failed to build query: %v", err.Error()))
				continue
			}
			err = es.Search(context.Background(), kafka.SourceResourcesSummaryIndex, query, &response)
			if err != nil {
				errs = append(errs, fmt.Sprintf("failed to run query: %v", err.Error()))
				continue
			}

			if len(response.Hits.Hits) > 0 {
				// there will be only one result anyway
				switch idx {
				case 0:
					lastDayValue = response.Hits.Hits[0].Source.ResourceCount
				case 1:
					lastWeekValue = response.Hits.Hits[0].Source.ResourceCount
				case 2:
					lastQuarterValue = response.Hits.Hits[0].Source.ResourceCount
				case 3:
					lastYearValue = response.Hits.Hits[0].Source.ResourceCount
				}
			}
		}

		msgs = append(msgs, kafka.SourceCategorySummary{
			CategoryName:     name,
			SourceType:       job.SourceType,
			SourceJobID:      job.JobID,
			DescribedAt:      job.DescribedAt,
			ResourceCount:    count,
			LastDayCount:     lastDayValue,
			LastWeekCount:    lastWeekValue,
			LastQuarterCount: lastQuarterValue,
			LastYearCount:    lastYearValue,
			ReportType:       kafka.ResourceSummaryTypeLastCategorySummary,
		})

		msgs = append(msgs, kafka.SourceCategorySummary{
			CategoryName:     name,
			SourceType:       job.SourceType,
			SourceJobID:      job.JobID,
			DescribedAt:      job.DescribedAt,
			ResourceCount:    count,
			LastDayCount:     lastDayValue,
			LastWeekCount:    lastWeekValue,
			LastQuarterCount: lastQuarterValue,
			LastYearCount:    lastYearValue,
			ReportType:       kafka.ResourceSummaryTypeCategoryHistorySummary,
		})
	}

	trend := kafka.SourceResourcesSummary{
		SourceID:      job.SourceID,
		SourceType:    job.SourceType,
		SourceJobID:   job.JobID,
		DescribedAt:   job.DescribedAt,
		ResourceCount: len(output.Resources),
		ReportType:    kafka.ResourceSummaryTypeResourceGrowthTrend,
	}
	msgs = append(msgs, trend)

	last := kafka.SourceResourcesLastSummary{
		SourceResourcesSummary: trend,
	}
	last.ReportType = kafka.ResourceSummaryTypeLastSummary
	msgs = append(msgs, last)

	locDistribution := kafka.LocationDistributionResource{
		SourceID:             job.SourceID,
		SourceType:           job.SourceType,
		SourceJobID:          job.JobID,
		LocationDistribution: locationDistribution,
		ReportType:           kafka.ResourceSummaryTypeLocationDistribution,
	}
	msgs = append(msgs, locDistribution)

	for serviceName, m := range serviceDistribution {
		msgs = append(msgs, kafka.SourceServiceDistributionResource{
			SourceID:             job.SourceID,
			ServiceName:          serviceName,
			SourceType:           job.SourceType,
			SourceJobID:          job.JobID,
			LocationDistribution: m,
			ReportType:           kafka.ResourceSummaryTypeServiceDistributionSummary,
		})
	}

	// For AWS resources, since they are queries independently per region,
	// if there is an error in some regions, return those errors. For the regions
	// with no error, return the list of resources.
	if len(errs) > 0 {
		err = fmt.Errorf("AWS: [%s]", strings.Join(errs, ","))
	} else {
		err = nil
	}

	return msgs, err
}

func doDescribeAzure(ctx context.Context, es keibi.Client, job DescribeJob, config map[string]interface{}) ([]kafka.DescribedResource, error) {
	creds, err := AzureSubscriptionConfigFromMap(config)
	if err != nil {
		return nil, fmt.Errorf("aure subscription credentials: %w", err)
	}

	subscriptionId := job.AccountID
	if len(subscriptionId) == 0 {
		subscriptionId = creds.SubscriptionID
	}

	output, err := azure.GetResources(
		ctx,
		job.ResourceType,
		[]string{subscriptionId},
		azure.AuthConfig{
			TenantID:            creds.TenantID,
			ClientID:            creds.ClientID,
			ClientSecret:        creds.ClientSecret,
			CertificatePath:     creds.CertificatePath,
			CertificatePassword: creds.CertificatePass,
			Username:            creds.Username,
			Password:            creds.Password,
		},
		string(azure.AuthEnv),
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("azure: %w", err)
	}

	serviceDistribution := map[string]map[string]int{}
	categoryCount := map[string]int{}
	serviceCount := map[string]int{}
	var msgs []kafka.DescribedResource
	locationDistribution := map[string]int{}
	for _, resource := range output.Resources {
		if resource.Description == nil {
			continue
		}

		msgs = append(msgs, kafka.Resource{
			ID:            resource.UniqueID(),
			Description:   resource.Description,
			SourceType:    api.SourceCloudAzure,
			ResourceType:  output.Metadata.ResourceType,
			ResourceJobID: job.JobID,
			SourceJobID:   job.ParentJobID,
			SourceID:      job.SourceID,
			Metadata: map[string]string{
				"id":                resource.ID,
				"name":              resource.Name,
				"subscription_id":   strings.Join(output.Metadata.SubscriptionIds, ","),
				"location":          resource.Location,
				"cloud_environment": output.Metadata.CloudEnvironment,
				"resource_type":     resource.Type,
			},
		})

		msgs = append(msgs, kafka.LookupResource{
			ResourceID:    resource.UniqueID(),
			Name:          resource.Name,
			SourceType:    api.SourceCloudAzure,
			ResourceType:  job.ResourceType,
			ResourceGroup: resource.ResourceGroup,
			Location:      resource.Location,
			SourceID:      job.SourceID,
			ResourceJobID: job.JobID,
			SourceJobID:   job.ParentJobID,
			CreatedAt:     job.DescribedAt,
			IsCommon:      cloudservice.IsCommonByResourceType(job.ResourceType),
		})
		location := strings.TrimSpace(resource.Location)
		if location != "" {
			locationDistribution[location]++
			if s := cloudservice.ServiceNameByResourceType(resource.Type); s != "" {
				if serviceDistribution[s] == nil {
					serviceDistribution[s] = make(map[string]int)
				}

				serviceDistribution[s][location]++
			}
		}
		if s := cloudservice.ServiceNameByResourceType(resource.Type); s != "" {
			serviceCount[s]++
		}
		if s := cloudservice.CategoryByResourceType(resource.Type); s != "" {
			if cloudservice.IsCommonByResourceType(resource.Type) {
				categoryCount[s]++
			}
		}
	}

	for name, count := range serviceCount {
		var lastDayValue, lastWeekValue, lastQuarterValue, lastYearValue int
		for idx, jobID := range []uint{job.LastDaySourceJobID, job.LastWeekSourceJobID, job.LastQuarterSourceJobID,
			job.LastYearSourceJobID} {
			var response ServiceQueryResponse
			query, err := FindOldServiceValue(jobID, name)
			if err != nil {
				return nil, fmt.Errorf("failed to build query: %v", err.Error())
			}
			err = es.Search(context.Background(), kafka.SourceResourcesSummaryIndex, query, &response)
			if err != nil {
				return nil, fmt.Errorf("failed to run query: %v", err.Error())
			}

			if len(response.Hits.Hits) > 0 {
				// there will be only one result anyway
				switch idx {
				case 0:
					lastDayValue = response.Hits.Hits[0].Source.ResourceCount
				case 1:
					lastWeekValue = response.Hits.Hits[0].Source.ResourceCount
				case 2:
					lastQuarterValue = response.Hits.Hits[0].Source.ResourceCount
				case 3:
					lastYearValue = response.Hits.Hits[0].Source.ResourceCount
				}
			}
		}

		msgs = append(msgs, kafka.SourceServicesSummary{
			ServiceName:      name,
			SourceType:       job.SourceType,
			SourceJobID:      job.JobID,
			DescribedAt:      job.DescribedAt,
			ResourceCount:    count,
			LastDayCount:     0,
			LastWeekCount:    0,
			LastQuarterCount: 0,
			LastYearCount:    0,
			ReportType:       kafka.ResourceSummaryTypeLastServiceSummary,
		})

		msgs = append(msgs, kafka.SourceServicesSummary{
			ServiceName:      name,
			SourceType:       job.SourceType,
			SourceJobID:      job.JobID,
			DescribedAt:      job.DescribedAt,
			ResourceCount:    count,
			LastDayCount:     lastDayValue,
			LastWeekCount:    lastWeekValue,
			LastQuarterCount: lastQuarterValue,
			LastYearCount:    lastYearValue,
			ReportType:       kafka.ResourceSummaryTypeServiceHistorySummary,
		})
	}

	for name, count := range categoryCount {
		var lastDayValue, lastWeekValue, lastQuarterValue, lastYearValue int
		for idx, jobID := range []uint{job.LastDaySourceJobID, job.LastWeekSourceJobID, job.LastQuarterSourceJobID,
			job.LastYearSourceJobID} {
			var response ServiceQueryResponse
			query, err := FindOldServiceValue(jobID, name)
			if err != nil {
				return nil, fmt.Errorf("failed to build query: %v", err.Error())
			}
			err = es.Search(context.Background(), kafka.SourceResourcesSummaryIndex, query, &response)
			if err != nil {
				return nil, fmt.Errorf("failed to run query: %v", err.Error())
			}

			if len(response.Hits.Hits) > 0 {
				// there will be only one result anyway
				switch idx {
				case 0:
					lastDayValue = response.Hits.Hits[0].Source.ResourceCount
				case 1:
					lastWeekValue = response.Hits.Hits[0].Source.ResourceCount
				case 2:
					lastQuarterValue = response.Hits.Hits[0].Source.ResourceCount
				case 3:
					lastYearValue = response.Hits.Hits[0].Source.ResourceCount
				}
			}
		}

		msgs = append(msgs, kafka.SourceCategorySummary{
			CategoryName:     name,
			SourceType:       job.SourceType,
			SourceJobID:      job.JobID,
			DescribedAt:      job.DescribedAt,
			ResourceCount:    count,
			LastDayCount:     lastDayValue,
			LastWeekCount:    lastWeekValue,
			LastQuarterCount: lastQuarterValue,
			LastYearCount:    lastYearValue,
			ReportType:       kafka.ResourceSummaryTypeLastCategorySummary,
		})

		msgs = append(msgs, kafka.SourceCategorySummary{
			CategoryName:     name,
			SourceType:       job.SourceType,
			SourceJobID:      job.JobID,
			DescribedAt:      job.DescribedAt,
			ResourceCount:    count,
			LastDayCount:     0,
			LastWeekCount:    0,
			LastQuarterCount: 0,
			LastYearCount:    0,
			ReportType:       kafka.ResourceSummaryTypeCategoryHistorySummary,
		})
	}

	for serviceName, m := range serviceDistribution {
		msgs = append(msgs, kafka.SourceServiceDistributionResource{
			SourceID:             job.SourceID,
			ServiceName:          serviceName,
			SourceType:           job.SourceType,
			SourceJobID:          job.JobID,
			LocationDistribution: m,
			ReportType:           kafka.ResourceSummaryTypeServiceDistributionSummary,
		})
	}

	trend := kafka.SourceResourcesSummary{
		SourceID:      job.SourceID,
		SourceType:    job.SourceType,
		SourceJobID:   job.JobID,
		DescribedAt:   job.DescribedAt,
		ResourceCount: len(output.Resources),
		ReportType:    kafka.ResourceSummaryTypeResourceGrowthTrend,
	}
	msgs = append(msgs, trend)

	last := kafka.SourceResourcesLastSummary{
		SourceResourcesSummary: trend,
	}
	last.ReportType = kafka.ResourceSummaryTypeLastSummary
	msgs = append(msgs, last)

	locDistribution := kafka.LocationDistributionResource{
		SourceID:             job.SourceID,
		SourceType:           job.SourceType,
		SourceJobID:          job.JobID,
		LocationDistribution: locationDistribution,
		ReportType:           kafka.ResourceSummaryTypeLocationDistribution,
	}
	msgs = append(msgs, locDistribution)

	return msgs, nil
}

func ResourceTypeToESIndex(t string) string {
	t = stopWordsRe.ReplaceAllString(t, "_")
	return strings.ToLower(t)
}

type DescribeCleanupJob struct {
	JobID        uint // DescribeResourceJob ID
	ResourceType string
}

func (j DescribeCleanupJob) Do(esClient *elasticsearch.Client) error {
	startTime := time.Now().Unix()
	ctx, cancel := context.WithTimeout(context.Background(), cleanupJobTimeout)
	defer cancel()

	rIndex := ResourceTypeToESIndex(j.ResourceType)
	fmt.Printf("Cleaning resources with resource_job_id of %d from index %s\n", j.JobID, rIndex)

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"resource_job_id": j.JobID,
			},
		},
	}

	// Delete the resources from both inventory_summary and resource specific index
	indices := []string{
		rIndex,
		InventorySummaryIndex,
	}

	resp, err := keibi.DeleteByQuery(ctx, esClient, indices, query,
		esClient.DeleteByQuery.WithRefresh(true),
		esClient.DeleteByQuery.WithConflicts("proceed"),
	)
	if err != nil {
		DoDescribeCleanupJobsDuration.WithLabelValues(j.ResourceType, "failure").Observe(float64(time.Now().Unix() - startTime))
		DoDescribeCleanupJobsCount.WithLabelValues(j.ResourceType, "failure").Inc()
		return err
	}

	if len(resp.Failures) != 0 {
		body, err := json.Marshal(resp)
		if err != nil {
			return err
		}

		DoDescribeCleanupJobsDuration.WithLabelValues(j.ResourceType, "failure").Observe(float64(time.Now().Unix() - startTime))
		DoDescribeCleanupJobsCount.WithLabelValues(j.ResourceType, "failure").Inc()
		return fmt.Errorf("elasticsearch: delete by query: %s", string(body))
	}

	fmt.Printf("Successfully delete %d resources of type %s\n", resp.Deleted, j.ResourceType)
	DoDescribeCleanupJobsDuration.WithLabelValues(j.ResourceType, "successful").Observe(float64(time.Now().Unix() - startTime))
	DoDescribeCleanupJobsCount.WithLabelValues(j.ResourceType, "successful").Inc()
	return nil
}
