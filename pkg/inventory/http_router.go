package inventory

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
)

// Context extends the echo.Context interface with custom APIs
type Context struct {
	echo.Context
}

func InitializeRouter(handler *HttpHandler) (*echo.Echo, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("new zap logger: %s", err)
	}

	e := httpserver.Register(logger, handler)

	// add middleware to extend the default context
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			cc := &Context{ctx}
			return next(cc)
		}
	})

	return e, nil
}
