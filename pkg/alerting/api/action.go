package api

type CreateActionReq struct {
	Method  string            `json:"method"`
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type Action struct {
	Id      uint              `json:"id"`
	Method  string            `json:"method"`
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type UpdateActionRequest struct {
	Method  *string           `json:"method"`
	Url     *string           `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    *string           `json:"body"`
}

type SlackRequest struct {
	ChannelName string `json:"channel_name"`
	Text        string `json:"text"`
}

type SlackInputs struct {
	SlackUrl    string `json:"slack_url"`
	ChannelName string `json:"channel_name"`
}

type JiraInputs struct {
	AtlassianDomain   string `json:"atlassian_domain"`
	AtlassianApiToken string `json:"atlassian_api_token"`
	Email             string `json:"email"`
	IssueTypeId       string `json:"issue_type_id"`
	ProjectId         string `json:"project_id"`
}

type JiraAndStackResponse struct {
	ActionId uint `json:"action_id"`
}
