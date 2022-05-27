package onboard

import (
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"gopkg.in/go-playground/validator.v9"
)

func InitializeRouter() *echo.Echo {
	e := echo.New()
	e.Logger.SetLevel(log.DEBUG) // TODO: change in prod
	e.Pre(middleware.RemoveTrailingSlash())
	p := prometheus.NewPrometheus("keibi_http", nil)
	p.Use(e)

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
