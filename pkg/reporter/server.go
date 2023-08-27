package reporter

import (
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"
)

var tracer = otel.Tracer("echo-server")

type HttpServer struct {
	Address string
	Logger  *zap.Logger
	Service *Service
}

func bindValidate(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}

	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
}

func NewHTTPServer(
	address string,
	logger *zap.Logger,
	j *Service,
) *HttpServer {
	return &HttpServer{
		Address: address,
		Logger:  logger,
		Service: j,
	}
}

func (h HttpServer) Register(e *echo.Echo) {
	e.POST("/job/trigger", h.TriggerQuery)
	e.GET("/job/:id", h.GetJob)
	e.GET("/jaeger/test", func(ctx echo.Context) error {
		//jaegertracing.TraceFunction(ctx, slowFunc, "Test String")
		return ctx.String(http.StatusOK, "Hello, World!")
	})
}

func slowFunc(s string) {
	time.Sleep(200 * time.Millisecond)
	return
}

func (h HttpServer) TriggerQuery(ctx echo.Context) error {
	var reqBody TriggerQueryRequest
	err := bindValidate(ctx, &reqBody)
	if err != nil {
		return err
	}

	//sp := jaegertracing.CreateChildSpan(ctx, "trigger query child span")
	//defer sp.Finish()
	//sp.LogKV("Test log")
	//sp.SetBaggageItem("Test baggage", "baggage")
	//sp.SetTag("Test tag", "New Tag")

	_, span := tracer.Start(ctx.Request().Context(), "job_trigger")
	defer span.End()

	var connectionId string
	if len(reqBody.Source) > 0 {
		connectionId = reqBody.Source
	} else {
		connection, err := h.Service.RandomAccount()
		if err != nil {
			return err
		}
		connectionId = connection.ID.String()
	}

	dbJob, err := h.Service.TriggerJob(connectionId, reqBody.Queries)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, dbJob)
}

func (h HttpServer) GetJob(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid job id")
	}
	dbJob, err := h.Service.db.GetWorkerJob(uint(id))
	if err != nil {
		h.Logger.Error("Error getting job", zap.Error(err))
		return err
	}
	return ctx.JSON(http.StatusOK, dbJob)
}
