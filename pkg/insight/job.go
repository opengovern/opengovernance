package insight

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/managedgrafana"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/go-errors/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	azuremodel "gitlab.com/keibiengine/keibi-engine/pkg/azure/model"

	awsmodel "gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
	"gitlab.com/keibiengine/keibi-engine/pkg/insight/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/insight/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
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
	JobID            uint
	QueryID          uint
	SmartQueryID     uint
	SourceID         string
	ScheduleJobUUID  string
	AccountID        string
	SourceType       source.Type
	Internal         bool
	Query            string
	Description      string
	Category         string
	ExecutedAt       int64
	LastDayJobID     uint
	LastWeekJobID    uint
	LastMonthJobID   uint
	LastQuarterJobID uint
	LastYearJobID    uint
}

type JobResult struct {
	JobID  uint
	Status api.InsightJobStatus
	Error  string
}

func (j Job) Do(client keibi.Client, steampipeConn *steampipe.Database, onboardClient client.OnboardServiceClient, producer sarama.SyncProducer, uploader *s3manager.Uploader, bucket, topic string, logger *zap.Logger) (r JobResult) {
	startTime := time.Now().Unix()
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("paniced with error:", err)
			fmt.Println(errors.Wrap(err, 2).ErrorStack())

			DoInsightJobsDuration.WithLabelValues(strconv.Itoa(int(j.QueryID)), "failure").Observe(float64(time.Now().Unix() - startTime))
			DoInsightJobsCount.WithLabelValues(strconv.Itoa(int(j.QueryID)), "failure").Inc()
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
		DoInsightJobsDuration.WithLabelValues(strconv.Itoa(int(j.QueryID)), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoInsightJobsCount.WithLabelValues(strconv.Itoa(int(j.QueryID)), "failure").Inc()
		status = api.InsightJobFailed
		if firstErr == nil {
			firstErr = err
		}
	}
	var count int64
	var (
		locationsMap   map[string]struct{}
		connectionsMap map[string]struct{}
	)
	var res *steampipe.Result
	var err error
	if strings.TrimSpace(j.Query) == "accounts_count" {
		var totalAccounts int64
		totalAccounts, err = onboardClient.CountSources(&httpclient.Context{
			UserRole: managedgrafana.RoleAdmin,
		}, j.SourceType)
		count = totalAccounts
		res = &steampipe.Result{
			Headers: []string{"count"},
			Data:    [][]any{{count}},
		}
	} else {
		sourceIdFilterWhereClause := fmt.Sprintf("keibi_account_id = '%s'", j.SourceID)
		if strings.HasPrefix(strings.ToLower(j.SourceID), "all:") {
			sourceIdFilterWhereClause = "1=1"
		}
		res, err = steampipeConn.QueryAll(strings.ReplaceAll(j.Query, "$SOURCEID_WHERE_CLAUSE", sourceIdFilterWhereClause))
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
								connectionsMap = make(map[string]struct{})
							}
							connectionsMap[metadata.AccountID] = struct{}{}
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
								connectionsMap = make(map[string]struct{})
							}
							connectionsMap[metadata.SubscriptionID] = struct{}{}
						}
						break
					}
				}
				break
			}
		}
	}
	if err == nil {
		objectName := fmt.Sprintf("%d-%d.out", j.QueryID, j.JobID)
		content, err := json.Marshal(res)
		if err == nil {
			result, err := uploader.Upload(&s3manager.UploadInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(objectName),
				Body:   strings.NewReader(string(content)),
			})
			if err == nil {
				var lastDayValue, lastWeekValue, lastMonthValue, lastQuarterValue, lastYearValue *int64
				for idx, jobID := range []uint{j.LastDayJobID, j.LastWeekJobID, j.LastMonthJobID, j.LastQuarterJobID, j.LastYearJobID} {
					var response ResultQueryResponse
					query, err := FindOldInsightValue(jobID, j.QueryID)
					if err != nil {
						fail(fmt.Errorf("failed to build query: %w", err))
					}
					err = client.Search(context.Background(), es.InsightsIndex, query, &response)
					if err != nil {
						fail(fmt.Errorf("failed to run query: %w", err))
					}

					if len(response.Hits.Hits) > 0 {
						// there will be only one result anyway
						switch idx {
						case 0:
							lastDayValue = &response.Hits.Hits[0].Source.Result
						case 1:
							lastWeekValue = &response.Hits.Hits[0].Source.Result
						case 2:
							lastMonthValue = &response.Hits.Hits[0].Source.Result
						case 3:
							lastQuarterValue = &response.Hits.Hits[0].Source.Result
						case 4:
							lastYearValue = &response.Hits.Hits[0].Source.Result
						}
					}
				}

				var locations []string = nil
				if locationsMap != nil {
					locations = make([]string, 0, len(locationsMap))
					for location := range locationsMap {
						locations = append(locations, location)
					}
				}
				var connections []string = nil
				if connectionsMap != nil {
					connections = make([]string, 0, len(connectionsMap))
					for connection := range connectionsMap {
						connections = append(connections, connection)
					}
				}

				var resources []kafka.Doc
				resourceTypeList := []es.InsightResourceType{es.InsightResourceHistory, es.InsightResourceLast}
				if strings.HasPrefix(strings.ToLower(j.SourceID), "all") {
					resourceTypeList = []es.InsightResourceType{es.InsightResourceProviderHistory, es.InsightResourceProviderLast}
				}
				for _, resourceType := range resourceTypeList {
					resources = append(resources, es.InsightResource{
						JobID:            j.JobID,
						QueryID:          j.QueryID,
						SmartQueryID:     j.SmartQueryID,
						Query:            j.Query,
						Internal:         j.Internal,
						Description:      j.Description,
						SourceID:         j.SourceID,
						AccountID:        j.AccountID,
						Provider:         j.SourceType,
						Category:         j.Category,
						ExecutedAt:       time.Now().UnixMilli(),
						ScheduleUUID:     j.ScheduleJobUUID,
						Result:           count,
						LastDayValue:     lastDayValue,
						LastWeekValue:    lastWeekValue,
						LastMonthValue:   lastMonthValue,
						LastQuarterValue: lastQuarterValue,
						LastYearValue:    lastYearValue,
						ResourceType:     resourceType,
						Locations:        locations,
						Connections:      connections,
						S3Location:       result.Location,
					})
				}
				if err := kafka.DoSend(producer, topic, resources, logger); err != nil {
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
		DoInsightJobsDuration.WithLabelValues(strconv.Itoa(int(j.QueryID)), "successful").Observe(float64(time.Now().Unix() - startTime))
		DoInsightJobsCount.WithLabelValues(strconv.Itoa(int(j.QueryID)), "successful").Inc()
	}

	return JobResult{
		JobID:  j.JobID,
		Status: status,
		Error:  errMsg,
	}
}
