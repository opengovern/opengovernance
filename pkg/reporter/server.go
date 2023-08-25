package reporter

import (
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type HttpServer struct {
	Address string
	Logger  *zap.Logger
	Job     *Job
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
	j *Job,
) *HttpServer {
	return &HttpServer{
		Address: address,
		Logger:  logger,
		Job:     j,
	}
}

func (h HttpServer) Register(e *echo.Echo) {
	e.POST("/query/trigger", h.TriggerQuery)
}

func NewJaegerTracer() (opentracing.Tracer, io.Closer, error) {
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		return nil, nil, err
	}
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		return nil, nil, err
	}
	return tracer, closer, nil
}

func (h HttpServer) TriggerQuery(ctx echo.Context) error {
	tracer, closer, err := NewJaegerTracer()
	if err != nil {
		return err
	}
	opentracing.SetGlobalTracer(tracer)
	span, _ := opentracing.StartSpanFromContext(ctx.Request().Context(), "Handle /trigger_query")
	defer span.Finish()
	var reqBody TriggerQueryRequest
	err = bindValidate(ctx, &reqBody)
	if err != nil {
		return err
	}

	var source *api.Connection
	if len(reqBody.Source) > 0 {
		source, err = h.Job.onboardClient.GetSource(&httpclient.Context{
			UserRole: apiAuth.AdminRole,
		}, reqBody.Source)
		if err != nil {
			return err
		}
	} else {
		source, err = h.Job.RandomAccount()
		if err != nil {
			return err
		}
	}

	err, response := h.Job.RunJob(source, &reqBody.Query)
	if err != nil {
		return err
	}
	err = closer.Close()
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, response)
}
