package insight

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/go-errors/errors"
	azuremodel "github.com/kaytu-io/kaytu-azure-describer/azure/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	awsmodel "github.com/kaytu-io/kaytu-aws-describer/aws/model"
	"github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	"github.com/kaytu-io/kaytu-engine/pkg/insight/es"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"

	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
)

var DoInsightJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "insight_worker",
	Name:      "do_insight_jobs_total",
	Help:      "Count of done insight jobs in insight-worker service",
}, []string{"queryid", "status"})

var DoInsightJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "keibi",
	Subsystem: "insight_worker",
	Name:      "do_insight_jobs_duration_seconds",
	Help:      "Duration of done insight jobs in insight-worker service",
	Buckets:   []float64{5, 60, 300, 600, 1800, 3600, 7200, 36000},
}, []string{"queryid", "status"})

type Job struct {
	JobID           uint
	InsightID       uint
	SourceID        string
	ScheduleJobUUID string
	AccountID       string
	SourceType      source.Type
	Internal        bool
	Query           string
	Description     string
	ExecutedAt      int64
	IsStack         bool
}

type JobResult struct {
	JobID  uint
	Status api.InsightJobStatus
	Error  string
}

func (j Job) Do(client keibi.Client, steampipeOption *steampipe.Option, onboardClient client.OnboardServiceClient, producer *confluent_kafka.Producer, uploader *s3manager.Uploader, bucket, topic string, logger *zap.Logger) (r JobResult) {
	startTime := time.Now().Unix()
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("paniced with error:", err)
			fmt.Println(errors.Wrap(err, 2).ErrorStack())

			DoInsightJobsDuration.WithLabelValues(strconv.Itoa(int(j.InsightID)), "failure").Observe(float64(time.Now().Unix() - startTime))
			DoInsightJobsCount.WithLabelValues(strconv.Itoa(int(j.InsightID)), "failure").Inc()
			r = JobResult{
				JobID:  j.JobID,
				Status: api.InsightJobFailed,
				Error:  fmt.Sprintf("paniced: %s", err),
			}
		}
	}()

	// Assume it succeeded unless it fails somewhere
	var (
		status         = api.InsightJobSucceeded
		firstErr error = nil
	)

	fail := func(err error) {
		DoInsightJobsDuration.WithLabelValues(strconv.Itoa(int(j.InsightID)), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoInsightJobsCount.WithLabelValues(strconv.Itoa(int(j.InsightID)), "failure").Inc()
		status = api.InsightJobFailed
		if firstErr == nil {
			firstErr = err
		}
	}
	var count int64
	var (
		locationsMap   map[string]struct{}
		connectionsMap map[string]string
	)
	ctx := &httpclient.Context{
		UserRole: authApi.AdminRole,
	}
	var err error
	var res *steampipe.Result
	if strings.TrimSpace(j.Query) == "accounts_count" {
		var totalAccounts int64
		totalAccounts, _ = onboardClient.CountSources(ctx, j.SourceType)
		count = totalAccounts
		res = &steampipe.Result{
			Headers: []string{"count"},
			Data:    [][]any{{count}},
		}
	} else {
		isAllConnectionsQuery := "FALSE"
		if strings.HasPrefix(strings.ToLower(j.SourceID), "all:") {
			isAllConnectionsQuery = "TRUE"
		}
		query := strings.ReplaceAll(j.Query, "$CONNECITON_ID", j.SourceID)
		query = strings.ReplaceAll(query, "$IS_ALL_CONNECTIONS_QUERY", isAllConnectionsQuery)
		if j.IsStack == true {
			steampipeOption.Host = fmt.Sprintf("%s-steampipe-service.%s.svc.cluster.local", j.SourceID, CurrentWorkspaceID)
		} else {
			steampipeOption.Host = SteampipeHost
		}
		steampipeConn, err := steampipe.NewSteampipeDatabase(*steampipeOption)
		if err != nil {
			fail(fmt.Errorf("failed to create steampipe connection: %w", err))
			return
		}
		fmt.Println("Initialized steampipe database: ", *steampipeConn)

		res, err = steampipeConn.QueryAll(query)
		if res != nil {
			count = int64(len(res.Data))
			for colNo, col := range res.Headers {
				if strings.ToLower(col) != "keibi_metadata" {
					continue
				}
				for _, row := range res.Data {
					for cellColNo, cell := range row {
						if cellColNo != colNo {
							continue
						}
						if cell == nil {
							continue
						}
						switch j.SourceType {
						case source.CloudAWS:
							var metadata awsmodel.Metadata
							err = json.Unmarshal([]byte(cell.(string)), &metadata)
							if err != nil {
								break
							}
							if locationsMap == nil {
								locationsMap = make(map[string]struct{})
							}
							locationsMap[metadata.Region] = struct{}{}
							if connectionsMap == nil {
								connectionsMap = make(map[string]string)
							}
							connectionsMap[metadata.SourceID] = metadata.AccountID
						case source.CloudAzure:
							var metadata azuremodel.Metadata
							err = json.Unmarshal([]byte(cell.(string)), &metadata)
							if err != nil {
								break
							}
							if locationsMap == nil {
								locationsMap = make(map[string]struct{})
							}
							locationsMap[metadata.Location] = struct{}{}
							if connectionsMap == nil {
								connectionsMap = make(map[string]string)
							}
							connectionsMap[metadata.SourceID] = metadata.SubscriptionID
						}
						break
					}
				}
				break
			}
		}
	}
	logger.Info("Got the results, uploading to s3")
	if err == nil {
		objectName := fmt.Sprintf("%d-%d.out", j.InsightID, j.JobID)
		content, err := json.Marshal(res)
		if err == nil {
			result, err := uploader.Upload(&s3manager.UploadInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(objectName),
				Body:   strings.NewReader(string(content)),
			})
			if err == nil {
				var locations []string = nil
				if locationsMap != nil {
					locations = make([]string, 0, len(locationsMap))
					for location := range locationsMap {
						locations = append(locations, location)
					}
				}
				var connections []es.InsightConnection = nil
				if connectionsMap != nil {
					connections = make([]es.InsightConnection, 0, len(connectionsMap))
					for connectionID, originalID := range connectionsMap {
						connections = append(connections, es.InsightConnection{
							ConnectionID: connectionID,
							OriginalID:   originalID,
						})
					}
				}

				var resources []kafka.Doc
				resourceTypeList := []es.InsightResourceType{es.InsightResourceHistory, es.InsightResourceLast}
				if strings.HasPrefix(strings.ToLower(j.SourceID), "all:") {
					resourceTypeList = []es.InsightResourceType{es.InsightResourceProviderHistory, es.InsightResourceProviderLast}
				}
				for _, resourceType := range resourceTypeList {
					resources = append(resources, es.InsightResource{
						JobID:               j.JobID,
						InsightID:           j.InsightID,
						Query:               j.Query,
						Internal:            j.Internal,
						Description:         j.Description,
						SourceID:            j.SourceID,
						AccountID:           j.AccountID,
						Provider:            j.SourceType,
						ExecutedAt:          time.Now().UnixMilli(),
						Result:              count,
						ResourceType:        resourceType,
						Locations:           locations,
						IncludedConnections: connections,
						S3Location:          result.Location,
					})
				}
				if err := kafka.DoSend(producer, topic, -1, resources, logger); err != nil {
					fail(fmt.Errorf("send to kafka: %w", err))
				}
			} else {
				fail(fmt.Errorf("uploading to s3: %w", err))
			}
		} else {
			fail(fmt.Errorf("building content: %w", err))
		}

	} else {
		fail(fmt.Errorf("describe resources: %w", err))
	}

	errMsg := ""
	if firstErr != nil {
		errMsg = firstErr.Error()
	}
	if status == api.InsightJobSucceeded {
		DoInsightJobsDuration.WithLabelValues(strconv.Itoa(int(j.InsightID)), "successful").Observe(float64(time.Now().Unix() - startTime))
		DoInsightJobsCount.WithLabelValues(strconv.Itoa(int(j.InsightID)), "successful").Inc()
	}

	return JobResult{
		JobID:  j.JobID,
		Status: status,
		Error:  errMsg,
	}
}
