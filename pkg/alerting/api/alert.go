package api

type SlackResponse struct {
	ChannelName string `json:"channel_name"`
	Text        string `json:"text"`
}

type SlackInputs struct {
	SlackUrl    string `json:"slack_url"`
	ChannelName string `json:"channel_name"`
	RuleId      int    `json:"rule_id"`
}
