package service

import (
	"context"

	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/integration/repository"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Source struct {
	tracer trace.Tracer
	repo   repository.Source
}

func (h Source) List(ctx context.Context, types []source.Type) ([]model.Source, error) {
	var (
		sources []model.Source
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
		span.SetName("list.without-types")

		sources, err = h.repo.List()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	return sources, nil
}
