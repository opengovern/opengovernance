package api

type GetCredsForJobRequest struct {
	SourceID string `json:"sourceId"`
}

type GetCredsForJobResponse struct {
	Credentials string `json:"creds"`
}
