package summarizer

import (
	"fmt"
	"strconv"
	"time"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-errors/errors"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/api"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/compliancebuilder"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var DoComplianceSummarizerJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "kaytu",
	Subsystem: "summarizer_worker",
	Name:      "do_compliance_summarizer_jobs_total",
	Help:      "Count of done summarizer jobs in summarizer-worker service",
}, []string{"queryid", "status"})

var DoComplianceSummarizerJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "kaytu",
	Subsystem: "summarizer_worker",
	Name:      "do_compliance_summarizer_jobs_duration_seconds",
	Help:      "Duration of done summarizer jobs in summarizer-worker service",
	Buckets:   []float64{5, 60, 300, 600, 1800, 3600, 7200, 36000},
}, []string{"queryid", "status"})

func (j SummarizeJob) DoComplianceSummarizer(client kaytu.Client, complianceClient client.ComplianceServiceClient, producer *confluent_kafka.Producer, topic string, logger *zap.Logger) (r SummarizeJobResult) {
	logger.Info("Starting compliance summarizing", zap.Int("jobID", int(j.JobID)))
	startTime := time.Now().Unix()
	defer func() {
		if err := recover(); err != nil {
			logger.Error(fmt.Sprintf("paniced with error: %v", err), zap.Int("jobID", int(j.JobID)))
			fmt.Println("paniced with error:", err)
			fmt.Println(errors.Wrap(err, 2).ErrorStack())

			r = SummarizeJobResult{
				JobID:   j.JobID,
				Status:  api.SummarizerJobFailed,
				Error:   fmt.Sprintf("paniced: %s", err),
				JobType: j.JobType,
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
		DoComplianceSummarizerJobsDuration.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceSummarizerJobsCount.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Inc()
		status = api.SummarizerJobFailed
		if firstErr == nil {
			firstErr = err
		}
	}

	var msgs []kafka.Doc
	builders := []compliancebuilder.Builder{
		compliancebuilder.NewBenchmarkSummaryBuilder(logger, j.JobID, client, complianceClient),
	}
	var searchAfter []any
	for {
		findings, err := es.FetchActiveFindings(client, searchAfter, es.EsFetchPageSize)
		if err != nil {
			fail(fmt.Errorf("Failed to fetch lookups: %v ", err))
			break
		}

		if len(findings.Hits.Hits) == 0 {
			break
		}

		logger.Info("got a batch of finding resources", zap.Int("count", len(findings.Hits.Hits)))
		for _, finding := range findings.Hits.Hits {
			for _, b := range builders {
				b.Process(finding.Source)
			}
			searchAfter = finding.Sort
		}
	}
	logger.Info("processed finding resources")

	for _, b := range builders {
		msgs = append(msgs, b.Build()...)
	}
	logger.Info("built messages", zap.Int("count", len(msgs)))

	for _, b := range builders {
		err := b.Cleanup(j.JobID)
		if err != nil {
			fail(fmt.Errorf("Failed to cleanup: %v ", err))
		}
	}
	logger.Info("cleanup done")

	if len(msgs) > 0 {
		for i := 0; i < len(msgs); i += MaxKafkaSendBatchSize {
			end := i + MaxKafkaSendBatchSize
			if end > len(msgs) {
				end = len(msgs)
			}
			err := kafka.DoSend(producer, topic, -1, msgs[i:end], logger)
			if err != nil {
				fail(fmt.Errorf("Failed to send to kafka: %v ", err))
			}
		}
		logger.Info("sent to kafka")
	}

	errMsg := ""
	if firstErr != nil {
		errMsg = firstErr.Error()
	}
	if status == api.SummarizerJobSucceeded {
		DoComplianceSummarizerJobsDuration.WithLabelValues(strconv.Itoa(int(j.JobID)), "successful").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceSummarizerJobsCount.WithLabelValues(strconv.Itoa(int(j.JobID)), "successful").Inc()
	}

	return SummarizeJobResult{
		JobID:   j.JobID,
		Status:  status,
		Error:   errMsg,
		JobType: j.JobType,
	}
}
