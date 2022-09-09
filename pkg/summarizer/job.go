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

type DescribeJob struct {
	ID       uint
	SourceID string

	LastDaySourceJobID     uint
	LastWeekSourceJobID    uint
	LastQuarterSourceJobID uint
	LastYearSourceJobID    uint
}

type Job struct {
	JobID              uint
	DescribeSourceJobs []DescribeJob
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
		DoSummarizerJobsDuration.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoSummarizerJobsCount.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Inc()
		status = api.SummarizerJobFailed
		if firstErr == nil {
			firstErr = err
		}
	}

	var msgs []kafka.SummaryDoc

	for _, dsj := range j.DescribeSourceJobs {
		logger.Info("Building resources summary", zap.Int("jobID", int(j.JobID)))
		msg, err := j.BuildResourcesSummary(client, dsj)
		logger.Info(fmt.Sprintf("BuildResourcesSummary:%v, %v", msg, err), zap.Int("jobID", int(j.JobID)))

		if err != nil {
			fail(err)
		} else {
			msgs = append(msgs, msg)
		}
	}

	res, err := j.BuildServicesSummary(client)
	if err != nil {
		fail(err)
	} else {
		msgs = append(msgs, res...)
	}

	if len(msgs) > 0 {
		err := kafka.DoSendToKafka(producer, topic, msgs, logger)
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

func (job Job) BuildResourcesSummary(client keibi.Client, j DescribeJob) (kafka.SummaryDoc, error) {
	hits, err := es.FetchResourceSummary(client, j.ID, &j.SourceID)
	if err != nil {
		return nil, err
	}

	summary := kafka.ConnectionResourcesSummary{
		SummarizerJobID: job.JobID,
		SourceID:        j.SourceID,
		SourceJobID:     j.ID,
	}
	for _, hit := range hits {
		summary.SourceType = source.Type(hit.SourceType)
		summary.DescribedAt = hit.DescribedAt
		summary.ReportType = hit.ReportType
		summary.ResourceCount += hit.ResourceCount
	}

	for _, jobID := range []uint{j.LastDaySourceJobID, j.LastWeekSourceJobID, j.LastQuarterSourceJobID, j.LastYearSourceJobID} {
		if jobID == 0 {
			continue
		}

		hits, err := es.FetchResourceSummary(client, jobID, &j.SourceID)
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

func (j Job) BuildServicesSummary(client keibi.Client) ([]kafka.SummaryDoc, error) {
	var sourceJobIDs []uint
	for _, dsj := range j.DescribeSourceJobs {
		sourceJobIDs = append(sourceJobIDs, dsj.ID)
	}
	hits, err := es.FetchServicesSummary(client, sourceJobIDs)
	if err != nil {
		return nil, err
	}

	summary := map[string]kafka.ConnectionServicesSummary{}
	for _, hit := range hits {
		if _, ok := summary[hit.ServiceName]; !ok {
			summary[hit.ServiceName] = kafka.ConnectionServicesSummary{
				ServiceName:  hit.ServiceName,
				ResourceType: hit.ResourceType,
				SourceType:   source.Type(hit.SourceType),
				DescribedAt:  hit.DescribedAt,
				ReportType:   hit.ReportType,
			}
		}

		v := summary[hit.ServiceName]
		v.ResourceCount += hit.ResourceCount
		summary[hit.ServiceName] = v
	}

	for _, lastDaysValue := range []uint{1, 7, 93, 428} {
		var sourceJobIDs []uint
		for _, dsj := range j.DescribeSourceJobs {
			var jobID uint = 0
			switch lastDaysValue {
			case 1:
				jobID = dsj.LastDaySourceJobID
			case 7:
				jobID = dsj.LastDaySourceJobID
			case 93:
				jobID = dsj.LastDaySourceJobID
			case 428:
				jobID = dsj.LastDaySourceJobID
			}

			if jobID == 0 {
				continue
			}
			sourceJobIDs = append(sourceJobIDs, jobID)
		}
		if len(sourceJobIDs) == 0 {
			continue
		}

		historicHits, err := es.FetchServicesSummary(client, sourceJobIDs)
		if err != nil {
			return nil, err
		}

		if len(historicHits) == 0 {
			continue
		}

		history := map[string]kafka.ConnectionServicesSummary{}
		for _, hit := range historicHits {
			if _, ok := history[hit.ServiceName]; !ok {
				history[hit.ServiceName] = kafka.ConnectionServicesSummary{
					ServiceName:  hit.ServiceName,
					ResourceType: hit.ResourceType,
					SourceType:   source.Type(hit.SourceType),
					DescribedAt:  hit.DescribedAt,
					ReportType:   hit.ReportType,
				}
			}

			v := history[hit.ServiceName]
			v.ResourceCount += hit.ResourceCount
			history[hit.ServiceName] = v
		}

		for k, v := range history {
			s := summary[k]
			switch lastDaysValue {
			case 1:
				s.LastDayCount = &v.ResourceCount
			case 7:
				s.LastWeekCount = &v.ResourceCount
			case 93:
				s.LastQuarterCount = &v.ResourceCount
			case 428:
				s.LastYearCount = &v.ResourceCount
			}
			summary[k] = s
		}
	}

	var summaryList []kafka.SummaryDoc
	for _, v := range summary {
		summaryList = append(summaryList, &v)
	}
	return summaryList, nil
}
