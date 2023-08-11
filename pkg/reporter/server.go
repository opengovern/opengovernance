package reporter

import (
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
)

type HttpServer struct {
	Address string
	Logger  *zap.Logger
}

func bindValidate(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}

	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
}

func NewHTTPServer(
	address string,
	logger *zap.Logger,
) *HttpServer {
	return &HttpServer{
		Address: address,
		Logger:  logger,
	}
}

func (h HttpServer) Register(e *echo.Echo) {
	e.GET("/query/trigger", httpserver.AuthorizeHandler(h.TriggerQuery, apiAuth.AdminRole))
}

func (h HttpServer) TriggerQuery(ctx echo.Context) error {
	var reqBody Query
	bindValidate(ctx, &reqBody)
	return ctx.NoContent(http.StatusOK)
}
