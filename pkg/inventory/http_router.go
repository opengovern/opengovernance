package inventory

import (
	echoPrometheus "github.com/globocom/echo-prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/go-playground/validator.v9"
)

// Context extends the echo.Context interface with custom APIs
type Context struct {
	echo.Context
}

func InitializeRouter() *echo.Echo {
	e := echo.New()
	e.Logger.SetLevel(log.DEBUG) // TODO: change in prod
	e.Pre(middleware.RemoveTrailingSlash())

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

	// add middleware to extend the default context
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			cc := &Context{ctx}
			return next(cc)
		}
	})

	e.Use(middleware.Logger())
	e.Validator = newValidator()

	return e
}

type Validator struct {
	validate *validator.Validate
}

func newValidator() *Validator {
	return &Validator{
		validate: validator.New(),
	}
}

func (v *Validator) Validate(i interface{}) error {
	return v.validate.Struct(i)
}
