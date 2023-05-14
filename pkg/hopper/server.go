package hopper

import (
	config2 "github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/describe"
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"gitlab.com/keibiengine/keibi-engine/pkg/config"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
	"net/http"

	"github.com/labstack/echo/v4"
)

func initRabbitQueue(cnf config.RabbitMQ, queueName string) (queue.Interface, error) {
	qCfg := queue.Config{}
	qCfg.Server.Username = cnf.Username
	qCfg.Server.Password = cnf.Password
	qCfg.Server.Host = cnf.Service
	qCfg.Server.Port = 5672
	qCfg.Queue.Name = queueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = "describe-scheduler"
	insightQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	return insightQueue, nil
}

type HopperConfig struct {
	HttpServer config.HttpServer
	RabbitMQ   config.RabbitMQ
}

type HttpServer struct {
	config           HopperConfig
	hopperAWSQueue   queue.Interface
	hopperAzureQueue queue.Interface
}

func (h *HttpServer) Run() error {
	config2.ReadFromEnv(h.config, nil)

	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}

	logger.Info("Initializing the scheduler")

	h.hopperAzureQueue, err = initRabbitQueue(h.config.RabbitMQ, "hopper-aws-queue")
	if err != nil {
		return err
	}

	h.hopperAWSQueue, err = initRabbitQueue(h.config.RabbitMQ, "hopper-azure-queue")
	if err != nil {
		return err
	}

	return httpserver.RegisterAndStart(logger, h.config.HttpServer.Address, h)
}

func (h HttpServer) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.PUT("/trigger/describer/:source_type", h.TriggerDescriber)
}

func (h HttpServer) TriggerDescriber(ctx echo.Context) error {
	sourceType := ctx.Param("source_type")

	var req describe.LambdaDescribeWorkerInput
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	switch sourceType {
	case "aws":
		err := h.hopperAWSQueue.Publish(req)
		if err != nil {
			return err
		}
		return ctx.NoContent(http.StatusOK)

	case "azure":
		err := h.hopperAzureQueue.Publish(req)
		if err != nil {
			return err
		}
		return ctx.NoContent(http.StatusOK)
	}
	return echo.NewHTTPError(http.StatusNotImplemented, "provider not implemented")
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
