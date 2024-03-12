package api

type SupersetDashboard struct {
	ID   string `json:"ID"`
	Name string `json:"Name"`
}

type GenerateSupersetDashboardTokenResponse struct {
	Token      string              `json:"token"`
	Dashboards []SupersetDashboard `json:"dashboards"`
}
