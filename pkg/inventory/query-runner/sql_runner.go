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
	//lastIdx := (req.Page.No - 1) * req.Page.Size

	direction := inventoryApi.DirectionType("")
	//orderBy := ""
	//if req.Sorts != nil && len(req.Sorts) > 0 {
	//	direction = req.Sorts[0].Direction
	//	orderBy = req.Sorts[0].Field
	//}
	//if len(req.Sorts) > 1 {
	//	return nil, errors.New("multiple sort items not supported")
	//}

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
