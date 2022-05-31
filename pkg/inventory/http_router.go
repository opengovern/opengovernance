package inventory

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"gitlab.com/keibiengine/keibi-engine/pkg/metrics"
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

	metrics.AddEchoMiddleware(e)

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
