package compliance

import (
	"context"
	"go.opentelemetry.io/otel/trace"
)

func (h *HttpHandler) GetBenchmarkTreeIDs(ctx context.Context, rootID string) ([]string, error) {
	ids := []string{rootID}
	_, span2 := tracer.Start(ctx, "new_GetBenchmark", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_GetBenchmark")

	root, err := h.db.GetBenchmark(rootID)
	if err != nil {
		return nil, err
	}
	span2.End()

	for _, child := range root.Children {
		cids, err := h.GetBenchmarkTreeIDs(ctx, child.ID)
		if err != nil {
			return nil, err
		}

		ids = append(ids, cids...)
	}
	return ids, nil
}
