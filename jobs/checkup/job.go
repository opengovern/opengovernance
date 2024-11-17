package checkup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	authAPI "github.com/opengovern/og-util/pkg/api"
	shared_entities "github.com/opengovern/og-util/pkg/api/shared-entities"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/opengovernance/jobs/checkup/config"
	authClient "github.com/opengovern/opengovernance/services/auth/client"
	metadataClient "github.com/opengovern/opengovernance/services/metadata/client"
	"golang.org/x/net/context"

	"github.com/go-errors/errors"
	"github.com/opengovern/opengovernance/jobs/checkup/api"
	"github.com/opengovern/opengovernance/services/integration/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var DoCheckupJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "opengovernance",
	Subsystem: "checkup_worker",
	Name:      "do_checkup_jobs_total",
	Help:      "Count of done checkup jobs in checkup-worker service",
}, []string{"queryid", "status"})

var DoCheckupJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "opengovernance",
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

func (j Job) Do(integrationClient client.IntegrationServiceClient, authClient authClient.AuthServiceClient,
	metadataClient metadataClient.MetadataServiceClient, logger *zap.Logger, config config.WorkerConfig) (r JobResult) {
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
	integrations, err := integrationClient.ListIntegrations(&httpclient.Context{
		UserRole: authAPI.EditorRole,
	}, nil)
	if err != nil {
		logger.Error("failed to get connections list from onboard service", zap.Error(err))
		fail(fmt.Errorf("failed to get connections list from onboard service: %w", err))
	} else {
		for _, integrationObj := range integrations.Integrations {
			if integrationObj.LastCheck != nil && integrationObj.LastCheck.Add(8*time.Hour).After(time.Now()) {
				logger.Info("skipping integration health check", zap.String("integration_id", integrationObj.IntegrationID))
				continue
			}
			logger.Info("checking integration health", zap.String("integration_id", integrationObj.IntegrationID))
			_, err := integrationClient.IntegrationHealthcheck(&httpclient.Context{
				UserRole: authAPI.EditorRole,
			}, integrationObj.IntegrationID)
			if err != nil {
				logger.Error("failed to check integration health", zap.String("integration_id", integrationObj.IntegrationID), zap.Error(err))
				fail(fmt.Errorf("failed to check source health %s: %w", integrationObj.IntegrationID, err))
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

	if config.DoTelemetry {
		j.SendTelemetry(context.Background(), logger, config, integrationClient, authClient, metadataClient)
	}

	return JobResult{
		JobID:  j.JobID,
		Status: status,
		Error:  errMsg,
	}
}

func (j *Job) SendTelemetry(ctx context.Context, logger *zap.Logger, workerConfig config.WorkerConfig,
	integrationClient client.IntegrationServiceClient, authClient authClient.AuthServiceClient, metadataClient metadataClient.MetadataServiceClient) {
	now := time.Now()

	httpCtx := httpclient.Context{Ctx: ctx, UserRole: authAPI.AdminRole}

	req := shared_entities.CspmUsageRequest{
		GatherTimestamp:      now,
		Hostname:             workerConfig.TelemetryHostname,
		IntegrationTypeCount: make(map[string]int),
	}

	integrations, err := integrationClient.ListIntegrations(&httpCtx, nil)
	if err != nil {
		logger.Error("failed to list sources", zap.Error(err))
		return
	}
	for _, integration := range integrations.Integrations {
		if _, ok := req.IntegrationTypeCount[integration.IntegrationType.String()]; !ok {
			req.IntegrationTypeCount[integration.IntegrationType.String()] = 0
		}
		req.IntegrationTypeCount[integration.IntegrationType.String()] += 1
	}

	users, err := authClient.ListUsers(&httpCtx)
	if err != nil {
		logger.Error("failed to list users", zap.Error(err))
		return
	}
	req.NumberOfUsers = int64(len(users))

	about, err := metadataClient.GetAbout(&httpCtx)
	if err != nil {
		logger.Error("failed to get about", zap.Error(err))
		return
	}
	req.InstallId = about.InstallID

	url := fmt.Sprintf("%s/api/v1/information/usage", workerConfig.TelemetryBaseURL)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		logger.Error("failed to marshal telemetry request", zap.Error(err))
		return
	}
	var resp any
	if statusCode, err := httpclient.DoRequest(httpCtx.Ctx, http.MethodPost, url, httpCtx.ToHeaders(), reqBytes, &resp); err != nil {
		logger.Error("failed to send telemetry", zap.Error(err), zap.Int("status_code", statusCode), zap.String("url", url), zap.Any("req", req), zap.Any("resp", resp))
		return
	}

	logger.Info("sent telemetry", zap.String("url", url))
}
