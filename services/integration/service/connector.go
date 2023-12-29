package service

import (
	"context"

	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/integration/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Connector struct {
	tracer trace.Tracer
	repo   repository.Connector
	logger *zap.Logger
}

func NewConnector(
	repo repository.Connector,
	logger *zap.Logger,
) Connector {
	return Connector{
		tracer: otel.GetTracerProvider().Tracer("integration.service.credential"),
		repo:   repo,
		logger: logger.Named("service").Named("credential"),
	}
}

func (c Connector) List(ctx context.Context) ([]model.Connector, error) {
	ctx, span := c.tracer.Start(ctx, "list-with-filters")
	defer span.End()

	connectors, err := c.repo.List(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)

		return nil, err
	}

	return connectors, nil
}
