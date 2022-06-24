package insight

import (
	"fmt"
	"strconv"
	"time"

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
	JobID      uint
	QueryID    uint
	Query      string
	ExecutedAt int64
}

type JobResult struct {
	JobID  uint
	Status api.InsightJobStatus
	Error  string
}

func (j Job) Do(steampipeConn *steampipe.Database, producer sarama.SyncProducer, topic string, logger *zap.Logger) (r JobResult) {

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

	res, err := steampipeConn.Count(j.Query)
	if err == nil {
		result := res.Data[0][0]
		if v, ok := result.(int); ok {
			resource := kafka.InsightResource{
				JobID:      j.JobID,
				QueryID:    j.QueryID,
				Query:      j.Query,
				ExecutedAt: time.Now().UnixMilli(),
				Result:     v,
			}
			if err := kafka.DoSendToKafka(producer, topic, []kafka.InsightResource{resource}, logger); err != nil {
				fail(fmt.Errorf("send to kafka: %w", err))
			}
		} else {
			fail(fmt.Errorf("result is not int"))
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
