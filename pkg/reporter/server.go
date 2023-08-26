package reporter

import (
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/labstack/echo-contrib/jaegertracing"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"time"
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
	e.GET("/jaeger/test", func(ctx echo.Context) error {
		jaegertracing.TraceFunction(ctx, slowFunc, "Test String")
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

	sp := jaegertracing.CreateChildSpan(ctx, "trigger query child span")
	defer sp.Finish()
	sp.LogKV("Test log")
	sp.SetBaggageItem("Test baggage", "baggage")
	sp.SetTag("Test tag", "New Tag")

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
	return ctx.JSON(http.StatusOK, response)
}
