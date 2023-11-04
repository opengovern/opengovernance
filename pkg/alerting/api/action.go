package api

type CreateActionReq struct {
	Name    string            `json:"name"`
	Method  string            `json:"method"`
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type Action struct {
	Id      uint              `json:"id"`
	Name    string            `json:"name"`
	Method  string            `json:"method"`
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type UpdateActionRequest struct {
	Name    *string           `json:"name"`
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
	Name        string `json:"name"`
	SlackUrl    string `json:"slack_url"`
	ChannelName string `json:"channel_name"`
}

type JiraInputs struct {
	Name              string `json:"name"`
	AtlassianDomain   string `json:"atlassian_domain"`
	AtlassianApiToken string `json:"atlassian_api_token"`
	Email             string `json:"email"`
	IssueTypeId       string `json:"issue_type_id"`
	ProjectId         string `json:"project_id"`
}

type JiraAndStackResponse struct {
	ActionId uint `json:"action_id"`
}
