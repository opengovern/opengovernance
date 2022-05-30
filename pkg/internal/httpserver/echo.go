package httpserver

import (
	"github.com/brpaz/echozap"
	echoPrometheus "github.com/globocom/echo-prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"
)

type Routes interface {
	Register(router *echo.Echo)
}

func Register(logger *zap.Logger, routes Routes) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Recover())
	e.Use(echozap.ZapLogger(logger))

	e.Use(echoPrometheus.MetricsMiddlewareWithConfig(echoPrometheus.Config{
		Namespace: "keibi",
		Subsystem: "http",
		Buckets: []float64{
			0.001, // 1ms
			0.01,  // 10ms
			0.1,   // 100 ms
			0.2,
			0.5,
			1.0,  // 1s
			10.0, // 10s
		},
		NormalizeHTTPStatus: true,
	}))
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	e.Pre(middleware.RemoveTrailingSlash())

	e.Validator = customValidator{
		validate: validator.New(),
	}

	routes.Register(e)

	return e
}

func RegisterAndStart(logger *zap.Logger, address string, routes Routes) error {
	e := Register(logger, routes)
	return e.Start(address)
}

type customValidator struct {
	validate *validator.Validate
}

func (v customValidator) Validate(i interface{}) error {
	return v.validate.Struct(i)
}
