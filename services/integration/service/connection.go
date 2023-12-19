package service

import (
	"context"

	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/integration/repository"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Connection struct {
	tracer trace.Tracer
	repo   repository.Connection
}

func NewConnection(repo repository.Connection) Connection {
	return Connection{
		tracer: otel.GetTracerProvider().Tracer("integration.service.sources"),
		repo:   repo,
	}
}

func (h Connection) List(ctx context.Context, types []source.Type) ([]model.Connection, error) {
	var (
		sources []model.Connection
		err     error
	)

	_, span := h.tracer.Start(ctx, "list", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	if len(types) > 0 {
		sources, err = h.repo.ListOfTypes(types)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.End()
	} else {
		sources, err = h.repo.List()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	return sources, nil
}
