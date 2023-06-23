package checkup

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"strconv"
	"time"

	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/client"

	"github.com/kaytu-io/kaytu-engine/pkg/checkup/api"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/go-errors/errors"
	"go.uber.org/zap"
)

var DoCheckupJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "checkup_worker",
	Name:      "do_checkup_jobs_total",
	Help:      "Count of done checkup jobs in checkup-worker service",
}, []string{"queryid", "status"})

var DoCheckupJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "keibi",
	Subsystem: "checkup_worker",
	Name:      "do_checkup_jobs_duration_seconds",
	Help:      "Duration of done checkup jobs in checkup-worker service",
	Buckets:   []float64{5, 60, 300, 600, 1800, 3600, 7200, 36000},
}, []string{"queryid", "status"})

type Job struct {
	JobID      uint
	ExecutedAt int64
}

type JobResult struct {
	JobID  uint
	Status api.CheckupJobStatus
	Error  string
}

func (j Job) Do(onboardClient client.OnboardServiceClient, logger *zap.Logger) (r JobResult) {
	startTime := time.Now().Unix()
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("paniced with error:", err)
			fmt.Println(errors.Wrap(err, 2).ErrorStack())

			DoCheckupJobsDuration.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Observe(float64(time.Now().Unix() - startTime))
			DoCheckupJobsCount.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Inc()
			r = JobResult{
				JobID:  j.JobID,
				Status: api.CheckupJobFailed,
				Error:  fmt.Sprintf("paniced: %s", err),
			}
		}
	}()

	// Assume it succeeded unless it fails somewhere
	var (
		status         = api.CheckupJobSucceeded
		firstErr error = nil
	)

	fail := func(err error) {
		DoCheckupJobsDuration.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoCheckupJobsCount.WithLabelValues(strconv.Itoa(int(j.JobID)), "failure").Inc()
		status = api.CheckupJobFailed
		if firstErr == nil {
			firstErr = err
		}
	}

	sources, err := onboardClient.ListSources(&httpclient.Context{
		UserRole: api2.EditorRole,
	}, nil)
	if err != nil {
		fail(fmt.Errorf("failed to get sources list from onboard service: %w", err))
	} else {
		for _, source := range sources {
			_, err := onboardClient.GetSourceHealthcheck(&httpclient.Context{
				UserRole: api2.EditorRole,
			}, source.ID.String())
			if err != nil {
				fail(fmt.Errorf("failed to check source health %s: %w", source.ID.String(), err))
			}
		}
	}

	errMsg := ""
	if firstErr != nil {
		errMsg = firstErr.Error()
	}
	if status == api.CheckupJobSucceeded {
		DoCheckupJobsDuration.WithLabelValues(strconv.Itoa(int(j.JobID)), "successful").Observe(float64(time.Now().Unix() - startTime))
		DoCheckupJobsCount.WithLabelValues(strconv.Itoa(int(j.JobID)), "successful").Inc()
	}

	return JobResult{
		JobID:  j.JobID,
		Status: status,
		Error:  errMsg,
	}
}
