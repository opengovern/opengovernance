package analytics

import (
	"fmt"
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	es2 "github.com/kaytu-io/kaytu-engine/pkg/analytics/es"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
	"reflect"
	"time"
)

type JobStatus string

const (
	JobCreated              JobStatus = "CREATED"
	JobInProgress           JobStatus = "IN_PROGRESS"
	JobCompletedWithFailure JobStatus = "COMPLETED_WITH_FAILURE"
	JobCompleted            JobStatus = "COMPLETED"
)

type Job struct {
	JobID uint
}

type JobResult struct {
	JobID  uint
	Status JobStatus
	Error  string
}

func (j *Job) Do(
	db db.Database,
	steampipeDB *steampipe.Database,
	kfkProducer *confluent_kafka.Producer,
	kfkTopic string,
	onboardClient onboardClient.OnboardServiceClient,
	logger *zap.Logger,
) JobResult {
	result := JobResult{
		JobID:  j.JobID,
		Status: JobCompleted,
		Error:  "",
	}

	if err := j.Run(db, steampipeDB, kfkProducer, kfkTopic, onboardClient, logger); err != nil {
		result.Error = err.Error()
		result.Status = JobCompletedWithFailure
	}
	return result
}

func (j *Job) Run(
	db db.Database,
	steampipeDB *steampipe.Database,
	kfkProducer *confluent_kafka.Producer,
	kfkTopic string,
	onboardClient onboardClient.OnboardServiceClient,
	logger *zap.Logger) error {
	startTime := time.Now()
	metrics, err := db.ListMetrics()
	if err != nil {
		return err
	}

	connectionCache := map[string]api.Connection{}

	for _, metric := range metrics {
		providerResultMap := map[string]es2.ConnectorMetricTrendSummary{}
		regionResultMap := map[string]es2.RegionMetricTrendSummary{}

		res, err := steampipeDB.QueryAll(metric.Query)
		if err != nil {
			return err
		}

		for _, record := range res.Data {
			if len(record) != 3 {
				return fmt.Errorf("invalid query: %s", metric.Query)
			}

			sourceID, ok := record[0].(string)
			if !ok {
				return fmt.Errorf("invalid format for sourceID: [%s] %v", reflect.TypeOf(record[0]), record[0])
			}
			region, ok := record[1].(string)
			if !ok {
				return fmt.Errorf("invalid format for region: [%s] %v", reflect.TypeOf(record[1]), record[1])
			}
			count, ok := record[2].(int64)
			if !ok {
				return fmt.Errorf("invalid format for count: [%s] %v", reflect.TypeOf(record[2]), record[2])
			}

			var conn *api.Connection
			if cached, ok := connectionCache[sourceID]; ok {
				conn = &cached
			} else {
				conn, err = onboardClient.GetSource(&httpclient.Context{UserRole: api2.AdminRole}, sourceID)
				if err != nil {
					return fmt.Errorf("GetSource id=%s err=%v", sourceID, err)
				}
			}

			if conn == nil {
				return fmt.Errorf("connection not found: %s", sourceID)
			}

			var msgs []kafka.Doc
			msgs = append(msgs, es2.ConnectionMetricTrendSummary{
				ConnectionID:  conn.ID,
				Connector:     conn.Connector,
				MetricID:      metric.ID,
				ResourceCount: int(count),
				EvaluatedAt:   startTime.UnixMilli(),
				ReportType:    es.MetricTrendConnectionSummary,
			})
			if err := kafka.DoSend(kfkProducer, kfkTopic, -1, msgs, logger); err != nil {
				return err
			}

			if v, ok := providerResultMap[conn.Connector.String()]; ok {
				v.ResourceCount += int(count)
				providerResultMap[conn.Connector.String()] = v
			} else {
				vn := es2.ConnectorMetricTrendSummary{
					Connector:     conn.Connector,
					EvaluatedAt:   startTime.UnixMilli(),
					MetricID:      metric.ID,
					ResourceCount: int(count),
					ReportType:    es.MetricTrendConnectorSummary,
				}
				providerResultMap[conn.Connector.String()] = vn
			}

			if v, ok := regionResultMap[region]; ok {
				v.ResourceCount += int(count)
				regionResultMap[region] = v
			} else {
				vn := es2.RegionMetricTrendSummary{
					Region:        region,
					EvaluatedAt:   startTime.UnixMilli(),
					MetricID:      metric.ID,
					ResourceCount: int(count),
					ReportType:    es.MetricTrendConnectorSummary,
				}
				regionResultMap[region] = vn
			}
		}

		for _, res := range providerResultMap {
			var msgs []kafka.Doc
			msgs = append(msgs, res)
			if err := kafka.DoSend(kfkProducer, kfkTopic, -1, msgs, logger); err != nil {
				return err
			}
		}

		for _, res := range regionResultMap {
			var msgs []kafka.Doc
			msgs = append(msgs, res)
			if err := kafka.DoSend(kfkProducer, kfkTopic, -1, msgs, logger); err != nil {
				return err
			}
		}
	}

	return nil
}
