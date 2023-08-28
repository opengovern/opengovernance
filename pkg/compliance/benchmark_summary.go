package compliance

import "context"

func (h *HttpHandler) GetBenchmarkTreeIDs(ctx context.Context, rootID string) ([]string, error) {
	ids := []string{rootID}
	_, spanGB := tracer.Start(ctx, "GetBenchmark")

	root, err := h.db.GetBenchmark(rootID)
	if err != nil {
		return nil, err
	}
	spanGB.End()

	for _, child := range root.Children {
		cids, err := h.GetBenchmarkTreeIDs(ctx, child.ID)
		if err != nil {
			return nil, err
		}

		ids = append(ids, cids...)
	}
	return ids, nil
}
