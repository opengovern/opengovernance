package summarizer

import (
	"fmt"
	"strconv"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/api"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/kafka"

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
	JobID       uint
	SourceID    string
	SourceJobID uint

	LastDaySourceJobID     uint
	LastWeekSourceJobID    uint
	LastQuarterSourceJobID uint
	LastYearSourceJobID    uint
}

type JobResult struct {
	JobID  uint
	Status api.SummarizerJobStatus
	Error  string
}

func (j Job) Do(client keibi.Client, producer sarama.SyncProducer, topic string, logger *zap.Logger) (r JobResult) {
	startTime := time.Now().Unix()
	defer func() {
		if err := recover(); err != nil {
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
		DoSummarizerJobsDuration.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoSummarizerJobsCount.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Inc()
		status = api.SummarizerJobFailed
		if firstErr == nil {
			firstErr = err
		}
	}

	var msgs []kafka.SummaryDoc

	msg, err := j.BuildResourcesSummary(client)
	if err != nil {
		fail(err)
	} else {
		msgs = append(msgs, msg)
	}

	if len(msgs) > 0 {
		err = kafka.DoSendToKafka(producer, topic, msgs, logger)
		if err != nil {
			fail(err)
		}
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

func (j Job) BuildResourcesSummary(client keibi.Client) (kafka.SummaryDoc, error) {
	hits, err := es.FetchResourceSummary(client, j.SourceJobID, nil, &j.SourceID, nil)
	if err != nil {
		return nil, err
	}

	var summary kafka.ConnectionResourcesSummary
	for _, hit := range hits {
		if summary.SourceID != "" {
			summary.ResourceCount += hit.ResourceCount
		} else {
			summary = kafka.ConnectionResourcesSummary{
				SummarizerJobID: j.JobID,
				SourceID:        hit.SourceID,
				SourceType:      source.Type(hit.SourceType),
				SourceJobID:     hit.SourceJobID,
				DescribedAt:     hit.DescribedAt,
				ResourceCount:   hit.ResourceCount,
				ReportType:      hit.ReportType,
			}
		}
	}

	for _, jobID := range []uint{j.LastDaySourceJobID, j.LastWeekSourceJobID, j.LastQuarterSourceJobID, j.LastYearSourceJobID} {
		hits, err := es.FetchResourceSummary(client, jobID, nil, &j.SourceID, nil)
		if err != nil {
			return nil, err
		}

		if len(hits) == 0 {
			continue
		}

		count := 0
		for _, hit := range hits {
			count += hit.ResourceCount
		}

		switch jobID {
		case j.LastDaySourceJobID:
			summary.LastDayCount = &count
		case j.LastWeekSourceJobID:
			summary.LastWeekCount = &count
		case j.LastQuarterSourceJobID:
			summary.LastQuarterCount = &count
		case j.LastYearSourceJobID:
			summary.LastYearCount = &count
		}
	}

	return &summary, nil
}
