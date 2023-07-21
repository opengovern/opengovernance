package analytics

import (
	"fmt"
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	es2 "github.com/kaytu-io/kaytu-engine/pkg/analytics/es"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
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
	steampipeDB *gorm.DB,
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
	steampipeDB *gorm.DB,
	kfkProducer *confluent_kafka.Producer,
	kfkTopic string,
	onboardClient onboardClient.OnboardServiceClient,
	logger *zap.Logger) error {
	startTime := time.Now()
	metrics, err := db.ListMetrics()
	if err != nil {
		return err
	}

	srcs, err := onboardClient.ListSources(&httpclient.Context{
		UserRole: api.AdminRole,
	}, nil)
	if err != nil {
		return err
	}

	for _, metric := range metrics {
		providerResultMap := map[string]es2.ConnectorMetricTrendSummary{}

		for _, src := range srcs {
			supportsConnector := false
			for _, connector := range metric.Connectors {
				if src.Connector.String() == connector {
					supportsConnector = true
				}
			}

			if !supportsConnector {
				continue
			}

			query := metric.Query
			query = strings.ReplaceAll(query, "${ACCOUNT_ID}", src.ConnectionID)
			query = strings.ReplaceAll(query, "${CONNECTION_ID}", src.ID.String())

			fmt.Println(query)
			var count int64
			row := steampipeDB.Exec(query).Row()
			if err = row.Scan(&count); err != nil {
				return err
			}

			var msgs []kafka.Doc
			msgs = append(msgs, es2.ConnectionMetricTrendSummary{
				ConnectionID:  src.ID,
				Connector:     src.Connector,
				MetricName:    metric.Name,
				ResourceCount: int(count),
				EvaluatedAt:   startTime.UnixMilli(),
				ReportType:    es.MetricTrendConnectionSummary,
			})
			if err := kafka.DoSend(kfkProducer, kfkTopic, -1, msgs, logger); err != nil {
				return err
			}

			if v, ok := providerResultMap[src.Connector.String()]; ok {
				v.ResourceCount += int(count)
				providerResultMap[src.Connector.String()] = v
			} else {
				vn := es2.ConnectorMetricTrendSummary{
					Connector:     src.Connector,
					EvaluatedAt:   startTime.UnixMilli(),
					MetricName:    metric.Name,
					ResourceCount: int(count),
					ReportType:    es.MetricTrendConnectorSummary,
				}
				providerResultMap[src.Connector.String()] = vn
			}
		}

		for _, res := range providerResultMap {
			var msgs []kafka.Doc
			msgs = append(msgs, res)
			if err := kafka.DoSend(kfkProducer, kfkTopic, -1, msgs, logger); err != nil {
				return err
			}
		}
	}

	return nil
}
