package pipedrive

type GetOrganizationDetailsResponse struct {
	Success bool         `json:"success"`
	Data    Organization `json:"data"`
}
