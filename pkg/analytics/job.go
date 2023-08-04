package analytics

import (
	"context"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es/resource"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es/spend"
	"reflect"
	"strings"
	"time"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
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
	dbc db.Database,
	steampipeDB *steampipe.Database,
	kfkProducer *confluent_kafka.Producer,
	kfkTopic string,
	onboardClient onboardClient.OnboardServiceClient,
	logger *zap.Logger) error {
	startTime := time.Now()
	metrics, err := dbc.ListMetrics()
	if err != nil {
		return err
	}

	connectionCache := map[string]api.Connection{}

	for _, metric := range metrics {
		tagMap := metric.GetTagsMap()
		tagValue := tagMap[db.MetricTypeKey]
		isAsset := true
		if tagValue != nil && len(tagValue) > 0 {
			if tagValue[0] == db.MetricTypeSpend {
				isAsset = false
			}
		}

		if isAsset {
			err = j.DoAssetMetric(
				steampipeDB,
				kfkProducer,
				kfkTopic,
				onboardClient,
				logger,
				metric,
				connectionCache,
				startTime,
			)
			if err != nil {
				return err
			}
		} else {
			yesterday := time.Now().UTC().AddDate(0, 0, -1)
			year, month, day := yesterday.Date()
			yesterdayStart := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
			yesterdayEnd := time.Date(year, month, day, 23, 59, 59, 0, time.UTC)

			err = j.DoSpendMetric(
				steampipeDB,
				kfkProducer,
				kfkTopic,
				onboardClient,
				logger,
				metric,
				connectionCache,
				yesterdayStart,
				yesterdayEnd,
			)
			if err != nil {
				return err
			}

			today := time.Now().UTC()
			year, month, day = today.Date()
			todayStart := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
			todayEnd := time.Date(year, month, day, 23, 59, 59, 0, time.UTC)

			err = j.DoSpendMetric(
				steampipeDB,
				kfkProducer,
				kfkTopic,
				onboardClient,
				logger,
				metric,
				connectionCache,
				todayStart,
				todayEnd,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (j *Job) DoAssetMetric(
	steampipeDB *steampipe.Database,
	kfkProducer *confluent_kafka.Producer,
	kfkTopic string,
	onboardClient onboardClient.OnboardServiceClient,
	logger *zap.Logger,
	metric db.AnalyticMetric,
	connectionCache map[string]api.Connection,
	startTime time.Time,
) error {
	connectionResultMap := map[string]resource.ConnectionMetricTrendSummary{}
	providerResultMap := map[string]resource.ConnectorMetricTrendSummary{}
	regionResultMap := map[string]resource.RegionMetricTrendSummary{}

	res, err := steampipeDB.QueryAll(context.TODO(), metric.Query)
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
				if strings.Contains(err.Error(), "source not found") {
					continue
				}
				return fmt.Errorf("GetSource id=%s err=%v", sourceID, err)
			}
			if conn == nil {
				return fmt.Errorf("connection not found: %s", sourceID)
			}

			connectionCache[sourceID] = *conn
		}

		if v, ok := connectionResultMap[conn.ID.String()]; ok {
			v.ResourceCount += int(count)
			connectionResultMap[conn.ID.String()] = v
		} else {
			vn := resource.ConnectionMetricTrendSummary{
				ConnectionID:  conn.ID,
				Connector:     conn.Connector,
				MetricID:      metric.ID,
				ResourceCount: int(count),
				EvaluatedAt:   startTime.UnixMilli(),
			}
			connectionResultMap[conn.ID.String()] = vn
		}

		if v, ok := providerResultMap[conn.Connector.String()]; ok {
			v.ResourceCount += int(count)
			providerResultMap[conn.Connector.String()] = v
		} else {
			vn := resource.ConnectorMetricTrendSummary{
				Connector:     conn.Connector,
				EvaluatedAt:   startTime.UnixMilli(),
				MetricID:      metric.ID,
				ResourceCount: int(count),
			}
			providerResultMap[conn.Connector.String()] = vn
		}

		regionKey := region + "-" + conn.ID.String()
		if v, ok := regionResultMap[regionKey]; ok {
			v.ResourceCount += int(count)
			regionResultMap[regionKey] = v
		} else {
			vn := resource.RegionMetricTrendSummary{
				Region:        region,
				ConnectionID:  conn.ID,
				Connector:     conn.Connector,
				EvaluatedAt:   startTime.UnixMilli(),
				MetricID:      metric.ID,
				ResourceCount: int(count),
			}
			regionResultMap[regionKey] = vn
		}
	}

	var msgs []kafka.Doc
	for _, item := range connectionResultMap {
		msgs = append(msgs, item)
	}
	for _, item := range providerResultMap {
		msgs = append(msgs, item)
	}
	for _, item := range regionResultMap {
		msgs = append(msgs, item)
	}
	if err := kafka.DoSend(kfkProducer, kfkTopic, -1, msgs, logger); err != nil {
		return err
	}

	fmt.Printf("Write %d region docs, %d provider docs, %d connection docs\n", len(regionResultMap), len(providerResultMap), len(connectionResultMap))
	return nil
}

func (j *Job) DoSpendMetric(
	steampipeDB *steampipe.Database,
	kfkProducer *confluent_kafka.Producer,
	kfkTopic string,
	onboardClient onboardClient.OnboardServiceClient,
	logger *zap.Logger,
	metric db.AnalyticMetric,
	connectionCache map[string]api.Connection,
	startTime time.Time,
	endTime time.Time,
) error {
	connectionResultMap := map[string]spend.ConnectionMetricTrendSummary{}
	providerResultMap := map[string]spend.ConnectorMetricTrendSummary{}

	query := metric.Query
	query = strings.ReplaceAll(query, "$startTime", fmt.Sprintf("%d", startTime.UnixMilli()))
	query = strings.ReplaceAll(query, "$endTime", fmt.Sprintf("%d", endTime.UnixMilli()))

	res, err := steampipeDB.QueryAll(context.TODO(), query)
	if err != nil {
		return err
	}

	for _, record := range res.Data {
		if len(record) != 2 {
			return fmt.Errorf("invalid query: %s", query)
		}

		connectionID, ok := record[0].(string)
		if !ok {
			return fmt.Errorf("invalid format for connectionID: [%s] %v", reflect.TypeOf(record[0]), record[0])
		}
		sum, ok := record[1].(float64)
		if !ok {
			return fmt.Errorf("invalid format for sum: [%s] %v", reflect.TypeOf(record[1]), record[1])
		}

		var conn *api.Connection
		if cached, ok := connectionCache[connectionID]; ok {
			conn = &cached
		} else {
			conn, err = onboardClient.GetSource(&httpclient.Context{UserRole: api2.AdminRole}, connectionID)
			if err != nil {
				if strings.Contains(err.Error(), "source not found") {
					continue
				}
				return fmt.Errorf("GetSource id=%s err=%v", connectionID, err)
			}
			if conn == nil {
				return fmt.Errorf("connection not found: %s", connectionID)
			}

			connectionCache[connectionID] = *conn
		}

		if v, ok := connectionResultMap[conn.ID.String()]; ok {
			v.CostValue += sum
			connectionResultMap[conn.ID.String()] = v
		} else {
			vn := spend.ConnectionMetricTrendSummary{
				ConnectionID: conn.ID,
				Connector:    conn.Connector,
				Date:         startTime.Format("2006-01-02"),
				MetricID:     metric.ID,
				CostValue:    sum,
				StartTime:    startTime.UnixMilli(),
				EndTime:      endTime.UnixMilli(),
			}
			connectionResultMap[conn.ID.String()] = vn
		}

		if v, ok := providerResultMap[conn.Connector.String()]; ok {
			v.CostValue += sum
			providerResultMap[conn.Connector.String()] = v
		} else {
			vn := spend.ConnectorMetricTrendSummary{
				Connector: conn.Connector,
				Date:      startTime.Format("2006-01-02"),
				MetricID:  metric.ID,
				CostValue: sum,
				StartTime: startTime.UnixMilli(),
				EndTime:   endTime.UnixMilli(),
			}
			providerResultMap[conn.Connector.String()] = vn
		}
	}

	var msgs []kafka.Doc
	for _, item := range connectionResultMap {
		msgs = append(msgs, item)
	}
	for _, item := range providerResultMap {
		msgs = append(msgs, item)
	}
	if err := kafka.DoSend(kfkProducer, kfkTopic, -1, msgs, logger); err != nil {
		return err
	}

	fmt.Printf("Write %d provider docs, %d connection docs\n", len(providerResultMap), len(connectionResultMap))
	return nil
}
