package summarizer

import (
	"fmt"
	"strconv"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/builder"

	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/api"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"

	"github.com/go-errors/errors"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
)

var DoSummarizerJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "summarizer_worker",
	Name:      "do_summarizer_jobs_total",
	Help:      "Count of done summarizer jobs in summarizer-worker service",
}, []string{"queryid", "status"})

var DoSummarizerJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "keibi",
	Subsystem: "summarizer_worker",
	Name:      "do_summarizer_jobs_duration_seconds",
	Help:      "Duration of done summarizer jobs in summarizer-worker service",
	Buckets:   []float64{5, 60, 300, 600, 1800, 3600, 7200, 36000},
}, []string{"queryid", "status"})

type Job struct {
	JobID         uint
	ScheduleJobID uint

	LastDayScheduleJobID     uint
	LastWeekScheduleJobID    uint
	LastQuarterScheduleJobID uint
	LastYearScheduleJobID    uint
}

type JobResult struct {
	JobID  uint
	Status api.SummarizerJobStatus
	Error  string
}

func (j Job) Do(client keibi.Client, producer sarama.SyncProducer, topic string, logger *zap.Logger) (r JobResult) {
	logger.Info("Starting summarizing", zap.Int("jobID", int(j.JobID)))
	startTime := time.Now().Unix()
	defer func() {
		if err := recover(); err != nil {
			logger.Error(fmt.Sprintf("paniced with error: %v", err), zap.Int("jobID", int(j.JobID)))
			fmt.Println("paniced with error:", err)
			fmt.Println(errors.Wrap(err, 2).ErrorStack())

			DoSummarizerJobsDuration.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Observe(float64(time.Now().Unix() - startTime))
			DoSummarizerJobsCount.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Inc()
			r = JobResult{
				JobID:  j.JobID,
				Status: api.SummarizerJobFailed,
				Error:  fmt.Sprintf("paniced: %s", err),
			}
		}
	}()

	// Assume it succeeded unless it fails somewhere
	var (
		status         = api.SummarizerJobSucceeded
		firstErr error = nil
	)

	fail := func(err error) {
		logger.Info("failed due to", zap.Error(err))
		DoSummarizerJobsDuration.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoSummarizerJobsCount.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Inc()
		status = api.SummarizerJobFailed
		if firstErr == nil {
			firstErr = err
		}
	}

	var msgs []kafka.Doc
	builders := []builder.Builder{
		builder.NewResourceSummaryBuilder(client, j.JobID),
		builder.NewTrendSummaryBuilder(client, j.JobID),
		builder.NewLocationSummaryBuilder(client, j.JobID),
		builder.NewResourceTypeSummaryBuilder(client, j.JobID),
		builder.NewServiceSummaryBuilder(client, j.JobID),
		builder.NewCategorySummaryBuilder(client, j.JobID),
		builder.NewServiceLocationSummaryBuilder(client, j.JobID),
	}
	var searchAfter []interface{}
	for {
		lookups, err := es.FetchLookupsByScheduleJobID(client, j.ScheduleJobID, searchAfter, es.EsFetchPageSize)
		if err != nil {
			fail(fmt.Errorf("Failed to fetch lookups: %v ", err))
			break
		}

		if len(lookups.Hits.Hits) == 0 {
			break
		}

		logger.Info("got a batch of lookup resources", zap.Int("count", len(lookups.Hits.Hits)))
		for _, lookup := range lookups.Hits.Hits {
			for _, b := range builders {
				b.Process(lookup.Source)
			}
			searchAfter = lookup.Sort
		}
	}
	logger.Info("processed lookup resources")
	for _, b := range builders {
		err := b.PopulateHistory(j.LastDayScheduleJobID, j.LastWeekScheduleJobID, j.LastQuarterScheduleJobID, j.LastYearScheduleJobID)
		if err != nil {
			fail(fmt.Errorf("Failed to populate history: %v ", err))
		}
	}
	logger.Info("history populated")
	for _, b := range builders {
		msgs = append(msgs, b.Build()...)
	}
	logger.Info("built messages", zap.Int("count", len(msgs)))
	for _, b := range builders {
		err := b.Cleanup(j.ScheduleJobID)
		if err != nil {
			fail(fmt.Errorf("Failed to cleanup: %v ", err))
		}
	}
	logger.Info("cleanup done")

	if len(msgs) > 0 {
		err := kafka.DoSend(producer, topic, msgs, logger)
		if err != nil {
			fail(fmt.Errorf("Failed to send to kafka: %v ", err))
		}
		logger.Info("sent to kafka")
	}

	errMsg := ""
	if firstErr != nil {
		errMsg = firstErr.Error()
	}
	if status == api.SummarizerJobSucceeded {
		DoSummarizerJobsDuration.WithLabelValues(strconv.Itoa(int(j.JobID)), "successful").Observe(float64(time.Now().Unix() - startTime))
		DoSummarizerJobsCount.WithLabelValues(strconv.Itoa(int(j.JobID)), "successful").Inc()
	}

	return JobResult{
		JobID:  j.JobID,
		Status: status,
		Error:  errMsg,
	}
}
