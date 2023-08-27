package httpserver

import (
	"context"
	"github.com/brpaz/echozap"
	"github.com/kaytu-io/kaytu-util/pkg/metrics"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"os"

	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"
)

var agentHost = os.Getenv("JAEGER_AGENT_HOST")
var serviceName = os.Getenv("JAEGER_SERVICE_NAME")

type Routes interface {
	Register(router *echo.Echo)
}

func Register(logger *zap.Logger, routes Routes) (*echo.Echo, *sdktrace.TracerProvider) {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Recover())
	e.Use(echozap.ZapLogger(logger))

	metrics.AddEchoMiddleware(e)

	e.Pre(middleware.RemoveTrailingSlash())

	tp, err := initTracer()
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	e.Validator = customValidator{
		validate: validator.New(),
	}

	routes.Register(e)

	return e, tp
}

func RegisterAndStart(logger *zap.Logger, address string, routes Routes) error {
	e, tp := Register(logger, routes)

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
		}
	}()
	e.Use(otelecho.Middleware(serviceName))

	return e.Start(address)
}

type customValidator struct {
	validate *validator.Validate
}

func (v customValidator) Validate(i interface{}) error {
	return v.validate.Struct(i)
}

func QueryArrayParam(ctx echo.Context, paramName string) []string {
	var values []string
	for k, v := range ctx.QueryParams() {
		if k == paramName || k == paramName+"[]" {
			values = append(values, v...)
		}
	}
	return values
}

func initTracer() (*sdktrace.TracerProvider, error) {
	exporter, err := jaeger.New(jaeger.WithAgentEndpoint(jaeger.WithAgentHost(agentHost)))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}
