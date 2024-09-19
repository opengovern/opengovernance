package query_runner

import (
	"context"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	inventoryApi "github.com/kaytu-io/open-governance/pkg/inventory/api"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func (w *Worker) RunSQLNamedQuery(ctx context.Context, query string) (*QueryResult, error) {
	var err error

	direction := inventoryApi.DirectionType("")

	for i := 0; i < 10; i++ {
		err = w.steampipeConn.Conn().Ping(ctx)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		w.logger.Error("failed to ping steampipe", zap.Error(err))
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	w.logger.Info("executing named query", zap.String("query", query))
	res, err := w.steampipeConn.Query(ctx, query, nil, nil, "", steampipe.DirectionType(direction))
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	resp := QueryResult{
		Headers: res.Headers,
		Result:  res.Data,
	}
	return &resp, nil
}