package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	dexApi "github.com/dexidp/dex/api/v2"
	"github.com/jackc/pgtype"
	api3 "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/og-util/pkg/httpserver"
	client4 "github.com/opengovern/opengovernance/pkg/compliance/client"
	client3 "github.com/opengovern/opengovernance/pkg/describe/client"
	inventoryApi "github.com/opengovern/opengovernance/pkg/inventory/api"
	client2 "github.com/opengovern/opengovernance/pkg/inventory/client"
	onboardApi "github.com/opengovern/opengovernance/pkg/onboard/api"
	"github.com/opengovern/opengovernance/pkg/onboard/client"
	model2 "github.com/opengovern/opengovernance/services/demo-importer/db/model"
	"github.com/opengovern/opengovernance/services/migrator/db/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/gorm"
	_ "gorm.io/gorm"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"net/url"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opengovern/opengovernance/pkg/metadata/api"
	"github.com/opengovern/opengovernance/pkg/metadata/internal/src"
	"github.com/opengovern/opengovernance/pkg/metadata/models"
)

func (h HttpHandler) Register(r *echo.Echo) {
	v1 := r.Group("/api/v1")

	filter := v1.Group("/filter")
	filter.POST("", httpserver.AuthorizeHandler(h.AddFilter, api3.ViewerRole))
	filter.GET("", httpserver.AuthorizeHandler(h.GetFilters, api3.ViewerRole))

	metadata := v1.Group("/metadata")
	metadata.GET("/:key", httpserver.AuthorizeHandler(h.GetConfigMetadata, api3.ViewerRole))
	metadata.POST("", httpserver.AuthorizeHandler(h.SetConfigMetadata, api3.AdminRole))

	queryParameter := v1.Group("/query_parameter")
	queryParameter.POST("", httpserver.AuthorizeHandler(h.SetQueryParameter, api3.AdminRole))
	queryParameter.GET("", httpserver.AuthorizeHandler(h.ListQueryParameters, api3.ViewerRole))

	v3 := r.Group("/api/v3")
	v3.PUT("/sample/purge", httpserver.AuthorizeHandler(h.PurgeSampleData, api3.ViewerRole))
	v3.PUT("/sample/sync", httpserver.AuthorizeHandler(h.SyncDemo, api3.ViewerRole))
	v3.PUT("/sample/loaded", httpserver.AuthorizeHandler(h.WorkspaceLoadedSampleData, api3.ViewerRole))
	v3.GET("/sample/sync/status", httpserver.AuthorizeHandler(h.GetSampleSyncStatus, api3.ViewerRole))
	v3.GET("/migration/status", httpserver.AuthorizeHandler(h.GetMigrationStatus, api3.ViewerRole))
	v3.GET("/configured/status", httpserver.AuthorizeHandler(h.GetConfiguredStatus, api3.ViewerRole))
	v3.PUT("/configured/set", httpserver.AuthorizeHandler(h.SetConfiguredStatus, api3.AdminRole))
	v3.PUT("/configured/unset", httpserver.AuthorizeHandler(h.UnsetConfiguredStatus, api3.ViewerRole))
	v3.GET("/about", httpserver.AuthorizeHandler(h.GetAbout, api3.ViewerRole))
	v3.GET("/vault/configured", httpserver.AuthorizeHandler(h.VaultConfigured, api3.ViewerRole))
}

var tracer = otel.Tracer("metadata")

func bindValidate(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}

	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
}

// GetConfigMetadata godoc
//
//	@Summary		Get key metadata
//	@Description	Returns the config metadata for the given key
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			key	path		string	true	"Key"
//	@Success		200	{object}	models.ConfigMetadata
//	@Router			/metadata/api/v1/metadata/{key} [get]
func (h HttpHandler) GetConfigMetadata(ctx echo.Context) error {
	key := ctx.Param("key")
	_, span := tracer.Start(ctx.Request().Context(), "new_GetConfigMetadata", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetConfigMetadata")

	metadata, err := src.GetConfigMetadata(h.db, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "config not found")
		}
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("key", key),
	))
	span.End()
	return ctx.JSON(http.StatusOK, metadata.GetCore())
}

// SetConfigMetadata godoc
//
//	@Summary		Set key metadata
//	@Description	Sets the config metadata for the given key
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			req	body	api.SetConfigMetadataRequest	true	"Request Body"
//	@Success		200
//	@Router			/metadata/api/v1/metadata [post]
func (h HttpHandler) SetConfigMetadata(ctx echo.Context) error {
	var req api.SetConfigMetadataRequest
	if err := bindValidate(ctx, &req); err != nil {
		return err
	}

	key, err := models.ParseMetadataKey(req.Key)
	if err != nil {
		return err
	}

	err = httpserver.RequireMinRole(ctx, key.GetMinAuthRole())
	if err != nil {
		return err
	}
	_, span := tracer.Start(ctx.Request().Context(), "new_SetConfigMetadata", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_SetConfigMetadata")

	err = src.SetConfigMetadata(h.db, key, req.Value)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("key", key.String()),
	))
	span.End()

	return ctx.JSON(http.StatusOK, nil)
}

// AddFilter godoc
//
//	@Summary	add filter
//	@Security	BearerToken
//	@Tags		metadata
//	@Produce	json
//	@Param		req	body	models.Filter	true	"Request Body"
//	@Success	200
//	@Router		/metadata/api/v1/filter [post]
func (h HttpHandler) AddFilter(ctx echo.Context) error {
	var req models.Filter
	if err := bindValidate(ctx, &req); err != nil {
		return err
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_AddFilter", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_AddFilter")

	err := h.db.AddFilter(models.Filter{Name: req.Name, KeyValue: req.KeyValue})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("name", req.Name),
	))
	span.End()
	return ctx.JSON(http.StatusOK, nil)
}

// GetFilters godoc
//
//	@Summary	list filters
//	@Security	BearerToken
//	@Tags		metadata
//	@Produce	json
//	@Success	200	{object}	[]models.Filter
//	@Router		/metadata/api/v1/filter [get]
func (h HttpHandler) GetFilters(ctx echo.Context) error {
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListFilters", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilters")

	filters, err := h.db.ListFilters()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil
	}
	span.End()
	return ctx.JSON(http.StatusOK, filters)
}

// SetQueryParameter godoc
//
//	@Summary		Set query parameter
//	@Description	Sets the query parameters from the request body
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			req	body	api.SetQueryParameterRequest	true	"Request Body"
//	@Success		200
//	@Router			/metadata/api/v1/query_parameter [post]
func (h HttpHandler) SetQueryParameter(ctx echo.Context) error {
	var req api.SetQueryParameterRequest
	if err := bindValidate(ctx, &req); err != nil {
		return err
	}

	if len(req.QueryParameters) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "no query parameters provided")
	}

	dbQueryParams := make([]*models.QueryParameter, 0, len(req.QueryParameters))
	for _, apiParam := range req.QueryParameters {
		//key, err := models.ParseQueryParameterKey(apiParam.Key)
		//if err != nil {
		//	return err
		//}
		dbParam := models.QueryParameterFromAPI(apiParam)
		dbParam.Key = apiParam.Key
		dbQueryParams = append(dbQueryParams, &dbParam)
	}

	_, span := tracer.Start(ctx.Request().Context(), "new_SetQueryParameter", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_SetQueryParameter")
	err := h.db.SetQueryParameters(dbQueryParams)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.logger.Error("error setting query parameters", zap.Error(err))
		return err
	}
	span.End()

	return ctx.JSON(http.StatusOK, nil)
}

// ListQueryParameters godoc
//
//	@Summary		List query parameters
//	@Description	Returns the list of query parameters
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Success		200	{object}	api.ListQueryParametersResponse
//	@Router			/metadata/api/v1/query_parameter [get]
func (h HttpHandler) ListQueryParameters(ctx echo.Context) error {
	_, span := tracer.Start(ctx.Request().Context(), "new_ListQueryParameters", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListQueryParameters")

	queryParams, err := h.db.GetQueryParameters()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.logger.Error("error getting query parameters", zap.Error(err))
		return err
	}
	span.End()

	result := api.ListQueryParametersResponse{
		QueryParameters: make([]api.QueryParameter, 0, len(queryParams)),
	}
	for _, dbParam := range queryParams {
		apiParam := dbParam.ToAPI()
		result.QueryParameters = append(result.QueryParameters, apiParam)
	}

	return ctx.JSON(http.StatusOK, result)
}

// PurgeSampleData godoc
//
//	@Summary		List all workspaces with owner id
//	@Description	Returns all workspaces with owner id
//	@Security		BearerToken
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/workspace/api/v3/sample/purge [put]
func (h HttpHandler) PurgeSampleData(c echo.Context) error {
	ctx := &httpclient.Context{UserRole: api3.AdminRole}

	loaded, err := h.SampleDataLoaded(c)
	if err != nil {
		return err
	}
	if loaded == false {
		return echo.NewHTTPError(http.StatusNotFound, "Workspace does not contain sample data")
	}

	schedulerURL := strings.ReplaceAll(h.cfg.Scheduler.BaseURL, "%NAMESPACE%", h.cfg.OpengovernanceNamespace)
	schedulerClient := client3.NewSchedulerServiceClient(schedulerURL)

	complianceURL := strings.ReplaceAll(h.cfg.Compliance.BaseURL, "%NAMESPACE%", h.cfg.OpengovernanceNamespace)
	complianceClient := client4.NewComplianceClient(complianceURL)

	onboardURL := strings.ReplaceAll(h.cfg.Onboard.BaseURL, "%NAMESPACE%", h.cfg.OpengovernanceNamespace)
	onboardClient := client.NewOnboardServiceClient(onboardURL)

	err = schedulerClient.PurgeSampleData(ctx)
	if err != nil {
		h.logger.Error("failed to purge scheduler data", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to purge scheduler data")
	}
	err = complianceClient.PurgeSampleData(ctx)
	if err != nil {
		h.logger.Error("failed to purge compliance data", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to purge compliance data")
	}
	err = onboardClient.PurgeSampleData(ctx)
	if err != nil {
		h.logger.Error("failed to purge onboard data", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to purge onboard data")
	}

	return c.NoContent(http.StatusOK)
}

// SyncDemo godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/workspace/api/v3/sample/sync [put]
func (h HttpHandler) SyncDemo(echoCtx echo.Context) error {
	ctx := echoCtx.Request().Context()

	var mig *model.Migration
	tx := h.migratorDb.ORM.Model(&model.Migration{}).Where("id = ?", model2.MigrationJobName).Find(&mig)
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		h.logger.Error("failed to get migration", zap.Error(tx.Error))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get migration")
	}

	if mig != nil && mig.ID == model2.MigrationJobName {
		h.logger.Info("last migration job", zap.Any("job", *mig))
		if mig.Status != "COMPLETED" && mig.UpdatedAt.After(time.Now().Add(-1*10*time.Minute)) {
			return echo.NewHTTPError(http.StatusBadRequest, "sync sample data already in progress")
		}
	}

	metadata, err := src.GetConfigMetadata(h.db, string(models.MetadataKeyCustomizationEnabled))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "config not found")
		}
		return err
	}

	cnf := metadata.GetCore()

	var enabled models.IConfigMetadata
	switch cnf.Type {
	case models.ConfigMetadataTypeString:
		enabled = &models.StringConfigMetadata{
			ConfigMetadata: cnf,
		}
	case models.ConfigMetadataTypeInt:
		intValue, err := strconv.ParseInt(cnf.Value, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "failed to parse int value")
		}
		enabled = &models.IntConfigMetadata{
			ConfigMetadata: cnf,
			Value:          int(intValue),
		}
	case models.ConfigMetadataTypeBool:
		boolValue, err := strconv.ParseBool(cnf.Value)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert bool to int")
		}
		enabled = &models.BoolConfigMetadata{
			ConfigMetadata: cnf,
			Value:          boolValue,
		}
	case models.ConfigMetadataTypeJSON:
		enabled = &models.JSONConfigMetadata{
			ConfigMetadata: cnf,
			Value:          cnf.Value,
		}
	}

	if !enabled.GetValue().(bool) {
		return echo.NewHTTPError(http.StatusForbidden, "customization is not allowed")
	}

	demoDataS3URL := echoCtx.QueryParam("demo_data_s3_url")
	if demoDataS3URL != "" {
		// validate url
		_, err := url.ParseRequestURI(demoDataS3URL)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid url")
		}
		err = src.SetConfigMetadata(h.db, models.DemoDataS3URL, demoDataS3URL)
		if err != nil {
			h.logger.Error("set config metadata", zap.Error(err))
			return err
		}
	}

	var importDemoJob batchv1.Job
	err = h.kubeClient.Get(ctx, k8sclient.ObjectKey{
		Namespace: h.cfg.OpengovernanceNamespace,
		Name:      "import-es-demo-data",
	}, &importDemoJob)
	if err != nil {
		return err
	}

	err = h.kubeClient.Delete(ctx, &importDemoJob)
	if err != nil {
		return err
	}

	for {
		err = h.kubeClient.Get(ctx, k8sclient.ObjectKey{
			Namespace: h.cfg.OpengovernanceNamespace,
			Name:      "import-es-demo-data",
		}, &importDemoJob)
		if err != nil {
			if k8sclient.IgnoreNotFound(err) == nil {
				break
			}
			return err
		}

		time.Sleep(1 * time.Second)
	}

	importDemoJob.ObjectMeta = metav1.ObjectMeta{
		Name:      "import-es-demo-data",
		Namespace: h.cfg.OpengovernanceNamespace,
		Annotations: map[string]string{
			"helm.sh/hook":        "post-install,post-upgrade",
			"helm.sh/hook-weight": "0",
		},
	}
	importDemoJob.Spec.Selector = nil
	importDemoJob.Spec.Suspend = aws.Bool(false)
	importDemoJob.Spec.Template.ObjectMeta = metav1.ObjectMeta{}
	importDemoJob.Status = batchv1.JobStatus{}

	err = h.kubeClient.Create(ctx, &importDemoJob)
	if err != nil {
		return err
	}

	var importDemoDbJob batchv1.Job
	err = h.kubeClient.Get(ctx, k8sclient.ObjectKey{
		Namespace: h.cfg.OpengovernanceNamespace,
		Name:      "import-psql-demo-data",
	}, &importDemoDbJob)
	if err != nil {
		return err
	}

	err = h.kubeClient.Delete(ctx, &importDemoDbJob)
	if err != nil {
		return err
	}

	for {
		err = h.kubeClient.Get(ctx, k8sclient.ObjectKey{
			Namespace: h.cfg.OpengovernanceNamespace,
			Name:      "import-psql-demo-data",
		}, &importDemoDbJob)
		if err != nil {
			if k8sclient.IgnoreNotFound(err) == nil {
				break
			}
			return err
		}

		time.Sleep(1 * time.Second)
	}

	importDemoDbJob.ObjectMeta = metav1.ObjectMeta{
		Name:      "import-psql-demo-data",
		Namespace: h.cfg.OpengovernanceNamespace,
		Annotations: map[string]string{
			"helm.sh/hook":        "post-install,post-upgrade",
			"helm.sh/hook-weight": "0",
		},
	}
	importDemoDbJob.Spec.Selector = nil
	importDemoDbJob.Spec.Suspend = aws.Bool(false)
	importDemoDbJob.Spec.Template.ObjectMeta = metav1.ObjectMeta{}
	importDemoDbJob.Status = batchv1.JobStatus{}

	err = h.kubeClient.Create(ctx, &importDemoDbJob)
	if err != nil {
		return err
	}

	jp := pgtype.JSONB{}
	err = jp.Set([]byte(""))
	if err != nil {
		return err
	}
	tx = h.migratorDb.ORM.Model(&model.Migration{}).Where("id = ?", model2.MigrationJobName).Update("status", "Started").Update("jobs_status", jp)
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		h.logger.Error("failed to update migration", zap.Error(tx.Error))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update migration")
	}

	return echoCtx.JSON(http.StatusOK, struct{}{})
}

// WorkspaceLoadedSampleData godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/workspace/api/v3/sample/loaded [put]
func (h HttpHandler) WorkspaceLoadedSampleData(echoCtx echo.Context) error {
	loaded, err := h.SampleDataLoaded(echoCtx)
	if err != nil {
		return err
	}

	if loaded {
		return echoCtx.String(http.StatusOK, "True")
	}
	return echoCtx.String(http.StatusOK, "False")
}

// GetMigrationStatus godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	api.GetMigrationStatusResponse
//	@Router			/workspace/api/v3/migration/status [get]
func (h HttpHandler) GetMigrationStatus(echoCtx echo.Context) error {
	var mig *model.Migration
	tx := h.migratorDb.ORM.Model(&model.Migration{}).Where("id = ?", "main").First(&mig)
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		h.logger.Error("failed to get migration", zap.Error(tx.Error))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get migration")
	}
	if mig == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "no migration job found")
	}
	jobsStatus := make(map[string]model.JobInfo)

	if len(mig.JobsStatus.Bytes) > 0 {
		err := json.Unmarshal(mig.JobsStatus.Bytes, &jobsStatus)
		if err != nil {
			return err
		}
	}

	var completedJobs int
	for _, status := range jobsStatus {
		if status.Status == model.JobStatusCompleted || status.Status == model.JobStatusFailed {
			completedJobs++
		}
	}

	var jobProgress float64
	if len(jobsStatus) > 0 {
		jobProgress = float64(completedJobs) / float64(len(jobsStatus))
	}
	return echoCtx.JSON(http.StatusOK, api.GetMigrationStatusResponse{
		Status:     mig.Status,
		JobsStatus: jobsStatus,
		Summary: struct {
			TotalJobs          int     `json:"total_jobs"`
			CompletedJobs      int     `json:"completed_jobs"`
			ProgressPercentage float64 `json:"progress_percentage"`
		}{
			TotalJobs:          len(jobsStatus),
			CompletedJobs:      completedJobs,
			ProgressPercentage: jobProgress * 100,
		},
	})
}

// GetSampleSyncStatus godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	api.GetSampleSyncStatusResponse
//	@Router			/workspace/api/v3/sample/sync/status [get]
func (h HttpHandler) GetSampleSyncStatus(echoCtx echo.Context) error {
	var mig *model.Migration
	tx := h.migratorDb.ORM.Model(&model.Migration{}).Where("id = ?", model2.MigrationJobName).First(&mig)
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		h.logger.Error("failed to get migration", zap.Error(tx.Error))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get migration")
	}
	var jobsStatus model2.ESImportProgress

	if len(mig.JobsStatus.Bytes) > 0 {
		err := json.Unmarshal(mig.JobsStatus.Bytes, &jobsStatus)
		if err != nil {
			return err
		}
	}
	return echoCtx.JSON(http.StatusOK, api.GetSampleSyncStatusResponse{
		Status:   mig.Status,
		Progress: jobsStatus.Progress,
	})
}

// GetConfiguredStatus godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/workspace/api/v3/configured/status [get]
func (h HttpHandler) GetConfiguredStatus(echoCtx echo.Context) error {
	appConfiguration, err := h.db.GetAppConfiguration()
	if err != nil {
		h.logger.Error("failed to get workspace", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get workspace")
	}

	if appConfiguration.Configured {
		return echoCtx.String(http.StatusOK, "True")
	} else {
		return echoCtx.String(http.StatusOK, "False")
	}
}

// SetConfiguredStatus godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/workspace/api/v3/configured/set [put]
func (h HttpHandler) SetConfiguredStatus(echoCtx echo.Context) error {
	err := h.db.AppConfigured(true)
	if err != nil {
		h.logger.Error("failed to set workspace configured", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to set workspace configured")
	}
	return echoCtx.NoContent(http.StatusOK)
}

// UnsetConfiguredStatus godoc
//
//	@Summary		Sync demo
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Param			demo_data_s3_url	query	string	false	"Demo Data S3 URL"
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/workspace/api/v3/configured/unset [put]
func (h HttpHandler) UnsetConfiguredStatus(echoCtx echo.Context) error {
	err := h.db.AppConfigured(false)
	if err != nil {
		h.logger.Error("failed to unset workspace configured", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to unset workspace configured")
	}
	return echoCtx.NoContent(http.StatusOK)
}

// GetAbout godoc
//
//	@Summary		Get About info
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	api.About
//	@Router			/workspace/api/v3/configured/status [put]
func (h HttpHandler) GetAbout(echoCtx echo.Context) error {
	ctx := httpclient.FromEchoContext(echoCtx)
	ctx.UserRole = api3.AdminRole

	version := ""
	var kaytuVersionConfig corev1.ConfigMap
	err := h.kubeClient.Get(echoCtx.Request().Context(), k8sclient.ObjectKey{
		Namespace: h.cfg.OpengovernanceNamespace,
		Name:      "kaytu-version",
	}, &kaytuVersionConfig)
	if err == nil {
		version = kaytuVersionConfig.Data["version"]
	} else {
		fmt.Printf("failed to load version due to %v\n", err)
	}

	onboardURL := strings.ReplaceAll(h.cfg.Onboard.BaseURL, "%NAMESPACE%", h.cfg.OpengovernanceNamespace)
	onboardClient := client.NewOnboardServiceClient(onboardURL)
	connections, err := onboardClient.ListSources(ctx, nil)

	integrations := make(map[string][]onboardApi.Connection)
	for _, c := range connections {
		if _, ok := integrations[c.Connector.String()]; !ok {
			integrations[c.Connector.String()] = make([]onboardApi.Connection, 0)
		}
		integrations[c.Connector.String()] = append(integrations[c.Connector.String()], c)
	}

	inventoryURL := strings.ReplaceAll(h.cfg.Inventory.BaseURL, "%NAMESPACE%", h.cfg.OpengovernanceNamespace)
	inventoryClient := client2.NewInventoryServiceClient(inventoryURL)

	var engine inventoryApi.QueryEngine
	engine = inventoryApi.QueryEngine_cloudql
	query := `SELECT
    (SELECT SUM(cost) FROM azure_costmanagement_costbyresourcetype) +
    (SELECT SUM(amortized_cost_amount) FROM aws_cost_by_service_daily) AS total_cost;`
	results, err := inventoryClient.RunQuery(ctx, inventoryApi.RunQueryRequest{
		Page: inventoryApi.Page{
			No:   1,
			Size: 1000,
		},
		Engine: &engine,
		Query:  &query,
		Sorts:  nil,
	})
	if err != nil {
		h.logger.Error("failed to run query", zap.Error(err))
	}

	var floatValue float64
	if results != nil {
		h.logger.Info("query result", zap.Any("result", results.Result))
		if len(results.Result) > 0 && len(results.Result[0]) > 0 {
			totalSpent := results.Result[0][0]
			floatValue, _ = totalSpent.(float64)
		}
	}

	var dexConnectors []api.DexConnectorInfo
	dexClient, err := newDexClient(h.cfg.DexGrpcAddr)
	if err != nil {
		h.logger.Error("failed to create dex client", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex client")
	}

	if dexClient != nil {
		dexRes, err := dexClient.ListConnectors(context.Background(), &dexApi.ListConnectorReq{})
		if err != nil {
			h.logger.Error("failed to list dex connectors", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to list dex connectors")
		}
		if dexRes != nil {
			for _, c := range dexRes.Connectors {
				dexConnectors = append(dexConnectors, api.DexConnectorInfo{
					ID:   c.Id,
					Name: c.Name,
					Type: c.Type,
				})
			}
		}
	}

	loaded, err := h.SampleDataLoaded(echoCtx)
	if err != nil {
		h.logger.Error("failed to load data", zap.Error(err))
	}
	response := api.About{
		DexConnectors:         dexConnectors,
		AppVersion:            version,
		WorkspaceCreationTime: time.Time{}, // TODO
		Users:                 nil,         // TODO
		PrimaryDomainURL:      h.cfg.PrimaryDomainURL,
		APIKeys:               nil, // TODO
		Integrations:          integrations,
		SampleData:            loaded,
		TotalSpendGoverned:    floatValue,
	}

	return echoCtx.JSON(http.StatusOK, response)
}

func newDexClient(hostAndPort string) (dexApi.DexClient, error) {
	conn, err := grpc.NewClient(hostAndPort, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("dial: %v", err)
	}
	return dexApi.NewDexClient(conn), nil
}

func (h HttpHandler) SampleDataLoaded(echoCtx echo.Context) (bool, error) {
	ctx := httpclient.FromEchoContext(echoCtx)
	ctx.UserRole = api3.AdminRole

	onboardURL := strings.ReplaceAll(h.cfg.Onboard.BaseURL, "%NAMESPACE%", h.cfg.OpengovernanceNamespace)
	onboardClient := client.NewOnboardServiceClient(onboardURL)

	connections, err := onboardClient.ListSources(ctx, nil)
	if err != nil {
		return false, echo.NewHTTPError(http.StatusInternalServerError, "failed to list integrations")
	}

	connectionChecks := strings.Split(h.cfg.SampledataIntegrationsCheck, ",")
	connectionsMap := make(map[string]bool)
	for _, c := range connectionChecks {
		connectionsMap[c] = true
	}

	credentials, err := onboardClient.ListCredentials(ctx, nil, nil, nil, 0, 0)
	if err != nil {
		return false, echo.NewHTTPError(http.StatusInternalServerError, "failed to list credentials")
	}
	credentialsMap := make(map[string]bool)
	for _, c := range credentials.Credentials {
		credentialsMap[c.ID] = true
	}

	for _, c := range connections {
		if _, ok := connectionsMap[c.ID.String()]; !ok {
			return false, nil
		} else {
			if _, ok2 := credentialsMap[c.CredentialID]; ok2 {
				return false, nil
			}
		}
	}
	if len(connections) == 0 {
		return false, nil
	}
	return true, nil
}

// VaultConfigured godoc
//
//	@Summary		Get About info
//
//	@Description	Syncs demo with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	api.About
//	@Router			/workspace/api/v3/vault/configured [get]
func (h HttpHandler) VaultConfigured(echoCtx echo.Context) error {

	return echoCtx.String(http.StatusOK, "True")
}
