package api

import (
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type API struct {
	q      queue.Interface
	logger *zap.Logger
}

func New(
	logger *zap.Logger,
	q queue.Interface,
) *API {
	return &API{
		logger: logger.Named("api"),
		q:      q,
	}
}

func (*API) Register(e *echo.Echo) {
}
