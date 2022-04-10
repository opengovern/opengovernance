package httpserver

import (
	"github.com/brpaz/echozap"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"
)

type Routes interface {
	Register(router *echo.Echo)
}

func Register(logger *zap.Logger, routes Routes) *echo.Echo {
	e := echo.New()
	e.Use(echozap.ZapLogger(logger))
	e.Pre(middleware.RemoveTrailingSlash())

	e.Validator = customValidator{
		v: validator.New(),
	}

	routes.Register(e)

	return e
}

func RegisterAndStart(logger *zap.Logger, address string, routes Routes) error {
	e := Register(logger, routes)
	return e.Start(address)
}

type customValidator struct {
	v *validator.Validate
}

func (v customValidator) Validate(i interface{}) error {
	return v.v.Struct(i)
}
