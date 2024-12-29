package api

type QueryParameter struct {
	Key           string `json:"key"`
	Value         string `json:"value"`
	ControlsCount int    `json:"controls_count"`
	QueriesCount  int    `json:"queries_count"`
}

type SetQueryParameterRequest struct {
	QueryParameters []QueryParameter `json:"query_parameters"`
}

type ListQueryParametersResponse struct {
	Items      []QueryParameter `json:"items"`
	TotalCount int              `json:"total_count"`
}
