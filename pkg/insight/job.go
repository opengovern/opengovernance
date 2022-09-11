package insight

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"github.com/aws/aws-sdk-go/service/managedgrafana"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"

	"github.com/jackc/pgtype"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"

	"gitlab.com/keibiengine/keibi-engine/pkg/insight/api"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gitlab.com/keibiengine/keibi-engine/pkg/insight/kafka"

	"github.com/go-errors/errors"
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
	Internal         bool
	Query            string
	Description      string
	Provider         string
	Category         string
	ExecutedAt       int64
	LastDayJobID     uint
	LastWeekJobID    uint
	LastQuarterJobID uint
	LastYearJobID    uint
}

type JobResult struct {
	JobID  uint
	Status api.InsightJobStatus
	Error  string
}

func (j Job) Do(es keibi.Client, steampipeConn *steampipe.Database, onboardClient client.OnboardServiceClient, producer sarama.SyncProducer, topic string, logger *zap.Logger) (r JobResult) {
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
	var res *steampipe.Result
	var err error
	if strings.TrimSpace(j.Query) == "accounts_count" {
		pr, _ := source.ParseType(j.Provider)
		var totalAccounts int64
		totalAccounts, err = onboardClient.CountSources(&httpclient.Context{
			UserRole: managedgrafana.RoleAdmin,
		}, pr)
		res.Data = [][]interface{}{{totalAccounts}}
	} else {
		res, err = steampipeConn.Count(j.Query)
	}
	if err == nil {
		result := res.Data[0][0]
		if v, ok := result.(pgtype.Numeric); ok {
			result = v.Int.Int64()
		}

		if v, ok := result.(int64); ok {
			var lastDayValue, lastWeekValue, lastQuarterValue, lastYearValue *int64
			for idx, jobID := range []uint{j.LastDayJobID, j.LastWeekJobID, j.LastQuarterJobID, j.LastYearJobID} {
				var response ResultQueryResponse
				query, err := FindOldInsightValue(jobID, j.QueryID)
				if err != nil {
					fail(fmt.Errorf("failed to build query: %w", err))
				}
				err = es.Search(context.Background(), kafka.InsightsIndex, query, &response)
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
						lastQuarterValue = &response.Hits.Hits[0].Source.Result
					case 3:
						lastYearValue = &response.Hits.Hits[0].Source.Result
					}
				}
			}

			var resources []kafka.InsightResource
			for _, resourceType := range []kafka.InsightResourceType{kafka.InsightResourceHistory, kafka.InsightResourceLast} {
				resources = append(resources, kafka.InsightResource{
					JobID:            j.JobID,
					QueryID:          j.QueryID,
					SmartQueryID:     j.SmartQueryID,
					Query:            j.Query,
					Internal:         j.Internal,
					Description:      j.Description,
					Provider:         j.Provider,
					Category:         j.Category,
					ExecutedAt:       time.Now().UnixMilli(),
					Result:           v,
					LastDayValue:     lastDayValue,
					LastWeekValue:    lastWeekValue,
					LastQuarterValue: lastQuarterValue,
					LastYearValue:    lastYearValue,
					ResourceType:     resourceType,
				})
			}
			if err := kafka.DoSendToKafka(producer, topic, resources, logger); err != nil {
				fail(fmt.Errorf("send to kafka: %w", err))
			}
		} else {
			fail(fmt.Errorf("result is not int: %v [%s]", result, reflect.TypeOf(result)))
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
