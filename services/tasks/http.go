package tasks

import (
	"crypto/rsa"
	"encoding/json"
	api2 "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/opencomply/services/tasks/api"
	"github.com/opengovern/opencomply/services/tasks/db"
	"github.com/opengovern/opencomply/services/tasks/db/models"
	"github.com/opengovern/opencomply/services/tasks/scheduler"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type httpRoutes struct {
	logger *zap.Logger

	platformPrivateKey *rsa.PrivateKey
	db                 db.Database
	mainScheduler      *scheduler.MainScheduler
}

func (r *httpRoutes) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")
	// Get all tasks
	v1.GET("/tasks", httpserver.AuthorizeHandler(r.getTasks, api2.EditorRole))
	// Create a new task
	v1.POST("/tasks/run", httpserver.AuthorizeHandler(r.runTask, api2.EditorRole))
	// Get Task Result
	v1.GET("/tasks/run/:id/result", httpserver.AuthorizeHandler(r.getTaskRunResult, api2.EditorRole))

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

func (r *httpRoutes) getTasks(ctx echo.Context) error {
	tasks, err := r.db.GetTaskList()
	if err != nil {
		r.logger.Error("failed to get tasks", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, "failed to get tasks")

	}

	return ctx.JSON(http.StatusOK, tasks)
}

func (r *httpRoutes) runTask(ctx echo.Context) error {
	var req api.RunTaskRequest
	if err := bindValidate(ctx, &req); err != nil {
		r.logger.Error("failed to bind task", zap.Error(err))
		return ctx.JSON(http.StatusBadRequest, "failed to bind task")
	}

	task, _ := r.db.GetTask(req.TaskID)
	if task == nil {
		r.logger.Error("failed to find task", zap.String("task", req.TaskID))
		return ctx.JSON(http.StatusInternalServerError, "failed to find task")
	}

	run := models.TaskRun{
		TaskID: req.TaskID,
		Status: models.TaskRunStatusCreated,
	}
	paramsJson, err := json.Marshal(req.Params)
	if err != nil {
		return err
	}
	err = run.Params.Set(paramsJson)
	if err != nil {
		r.logger.Error("failed to set params", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to set params")
	}

	if err := r.db.CreateTaskRun(&run); err != nil {
		r.logger.Error("failed to create task run", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, "failed to create task run")
	}

	return ctx.JSON(http.StatusCreated, run)
}

func (r *httpRoutes) getTaskRunResult(ctx echo.Context) error {
	id := ctx.Param("id")
	taskResults, err := r.db.GetTaskRunResult(id)
	if err != nil {
		r.logger.Error("failed to get task results", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, "failed to get task results")
	}

	return ctx.JSON(http.StatusOK, taskResults)
}
