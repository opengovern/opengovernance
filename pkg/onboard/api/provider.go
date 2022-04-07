package api

type ProviderState string

const (
	ProviderStateEnabled    = "enabled"
	ProviderStateComingSoon = "coming_soon"
	ProviderStateDisabled   = "disabled"
)

type Provider struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	State ProviderState `json:"state"`
	Type  string        `json:"type"`
}

type ProvidersResponse []Provider
