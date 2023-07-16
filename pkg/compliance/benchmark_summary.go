package compliance

func (h *HttpHandler) GetBenchmarkTreeIDs(rootID string) ([]string, error) {
	ids := []string{rootID}

	root, err := h.db.GetBenchmark(rootID)
	if err != nil {
		return nil, err
	}

	for _, child := range root.Children {
		cids, err := h.GetBenchmarkTreeIDs(child.ID)
		if err != nil {
			return nil, err
		}

		ids = append(ids, cids...)
	}
	return ids, nil
}
