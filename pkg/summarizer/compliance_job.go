package summarizer

import (
	"fmt"
	"strconv"
	"time"

	confluence_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"

	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/compliancebuilder"

	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"

	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/api"

	"github.com/go-errors/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var DoComplianceSummarizerJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "summarizer_worker",
	Name:      "do_compliance_summarizer_jobs_total",
	Help:      "Count of done summarizer jobs in summarizer-worker service",
}, []string{"queryid", "status"})

var DoComplianceSummarizerJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "keibi",
	Subsystem: "summarizer_worker",
	Name:      "do_compliance_summarizer_jobs_duration_seconds",
	Help:      "Duration of done summarizer jobs in summarizer-worker service",
	Buckets:   []float64{5, 60, 300, 600, 1800, 3600, 7200, 36000},
}, []string{"queryid", "status"})

type ComplianceJob struct {
	JobID         uint
	ScheduleJobID uint

	LastDayScheduleJobID     uint
	LastWeekScheduleJobID    uint
	LastQuarterScheduleJobID uint
	LastYearScheduleJobID    uint

	JobType JobType
}

type ComplianceJobResult struct {
	JobID  uint
	Status api.SummarizerJobStatus
	Error  string

	JobType JobType
}

func (j ComplianceJob) Do(client keibi.Client, producer *confluence_kafka.Producer, topic string, logger *zap.Logger) (r ComplianceJobResult) {
	logger.Info("Starting summarizing", zap.Int("jobID", int(j.JobID)))
	startTime := time.Now().Unix()
	defer func() {
		if err := recover(); err != nil {
			logger.Error(fmt.Sprintf("paniced with error: %v", err), zap.Int("jobID", int(j.JobID)))
			fmt.Println("paniced with error:", err)
			fmt.Println(errors.Wrap(err, 2).ErrorStack())

			DoComplianceSummarizerJobsDuration.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Observe(float64(time.Now().Unix() - startTime))
			DoComplianceSummarizerJobsCount.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Inc()
			r = ComplianceJobResult{
				JobID:   j.JobID,
				Status:  api.SummarizerJobFailed,
				Error:   fmt.Sprintf("paniced: %s", err),
				JobType: JobType_ComplianceSummarizer,
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
		compliancebuilder.NewBenchmarkSummaryBuilder(client, j.JobID),
		compliancebuilder.NewAlarmsBuilder(client, j.JobID),
		compliancebuilder.NewMetricsBuilder(client, j.JobID),
	}
	var searchAfter []interface{}
	for {
		findings, err := es.FetchFindingsByScheduleJobID(client, j.ScheduleJobID, searchAfter, es.EsFetchPageSize)
		if err != nil {
			fail(fmt.Errorf("Failed to fetch findings: %v ", err))
			break
		}

		if len(findings.Hits.Hits) == 0 {
			break
		}

		logger.Info("got a batch of findings resources", zap.Int("count", len(findings.Hits.Hits)))
		for _, finding := range findings.Hits.Hits {
			for _, b := range builders {
				err := b.Process(finding.Source)
				if err != nil {
					fail(fmt.Errorf("Failed to process due to: %v ", err))
				}
			}
			searchAfter = finding.Sort
		}
	}
	logger.Info("processed finding resources")
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
		err := kafka.DoSend(producer, topic, -1, msgs, logger)
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
		DoComplianceSummarizerJobsDuration.WithLabelValues(strconv.Itoa(int(j.JobID)), "successful").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceSummarizerJobsCount.WithLabelValues(strconv.Itoa(int(j.JobID)), "successful").Inc()
	}

	return ComplianceJobResult{
		JobID:   j.JobID,
		Status:  status,
		Error:   errMsg,
		JobType: JobType_ComplianceSummarizer,
	}
}
