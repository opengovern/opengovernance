package checkup

import (
	"fmt"
	"strconv"
	"time"

	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"

	"github.com/kaytu-io/kaytu-engine/pkg/checkup/api"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/go-errors/errors"
	"go.uber.org/zap"
)

var DoCheckupJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "kaytu",
	Subsystem: "checkup_worker",
	Name:      "do_checkup_jobs_total",
	Help:      "Count of done checkup jobs in checkup-worker service",
}, []string{"queryid", "status"})

var DoCheckupJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "kaytu",
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

	// Healthcheck
	logger.Info("starting healthcheck")
	connections, err := onboardClient.ListSources(&httpclient.Context{
		UserRole: api2.EditorRole,
	}, nil)
	if err != nil {
		logger.Error("failed to get connections list from onboard service", zap.Error(err))
		fail(fmt.Errorf("failed to get connections list from onboard service: %w", err))
	} else {
		for _, connectionObj := range connections {
			if connectionObj.LastHealthCheckTime.Add(8 * time.Hour).After(time.Now()) {
				logger.Info("skipping source health check", zap.String("source_id", connectionObj.ID.String()))
				continue
			}
			logger.Info("checking source health", zap.String("source_id", connectionObj.ID.String()))
			_, err := onboardClient.GetSourceHealthcheck(&httpclient.Context{
				UserRole: api2.EditorRole,
			}, connectionObj.ID.String(), true)
			if err != nil {
				logger.Error("failed to check source health", zap.String("source_id", connectionObj.ID.String()), zap.Error(err))
				fail(fmt.Errorf("failed to check source health %s: %w", connectionObj.ID.String(), err))
			}
		}
	}

	// Auto Onboard
	logger.Info("starting auto onboard")
	credentials, err := onboardClient.ListCredentials(&httpclient.Context{
		UserRole: api2.EditorRole,
	}, nil, nil, utils.GetPointer("healthy"), 10000, 1)
	if err != nil {
		logger.Error("failed to get credentials list from onboard service", zap.Error(err))
		fail(fmt.Errorf("failed to get credentials list from onboard service: %w", err))
	}
	for _, cred := range credentials.Credentials {
		if !cred.AutoOnboardEnabled {
			continue
		}
		logger.Info("triggering auto onboard", zap.String("credential_id", cred.ID))
		_, err := onboardClient.TriggerAutoOnboard(&httpclient.Context{
			UserRole: api2.EditorRole,
		}, cred.ID)
		if err != nil {
			logger.Error("failed to trigger auto onboard", zap.String("credential_id", cred.ID), zap.Error(err))
			fail(fmt.Errorf("failed to trigger auto onboard for credential %s: %w", cred.ID, err))
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
