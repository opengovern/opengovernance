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
		Subsystem: "inventory",
		Buckets: []float64{
			0.0005,
			0.001, // 1ms
			0.002,
			0.005,
			0.01, // 10ms
			0.02,
			0.05,
			0.1, // 100 ms
			0.2,
			0.5,
			1.0, // 1s
			2.0,
			5.0,
			10.0, // 10s
			15.0,
			20.0,
			30.0,
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
