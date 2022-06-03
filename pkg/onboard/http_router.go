package onboard

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
)

func InitializeRouter(handler *HttpHandler) (*echo.Echo, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("new zap logger: %s", err)
	}

	return httpserver.Register(logger, handler), nil
}
