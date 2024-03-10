package api

type Dashboard struct {
	ID   string `json:"ID"`
	Name string `json:"Name"`
}

type GenerateDashboardTokenResponse struct {
	Token      string      `json:"token"`
	Dashboards []Dashboard `json:"dashboards"`
}
