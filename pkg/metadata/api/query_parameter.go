package api

type QueryParameter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SetQueryParameterRequest struct {
	QueryParameters []QueryParameter `json:"queryParameters"`
}

type GetQueryParameterResponse struct {
	QueryParameter QueryParameter `json:"queryParameter"`
}

type ListQueryParametersResponse struct {
	QueryParameters []QueryParameter `json:"queryParameters"`
}
