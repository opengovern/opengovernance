package api

type ProviderTypeState string

const (
	ProviderTypeStateEnabled    = "enabled"
	ProviderTypeStateComingSoon = "coming_soon"
	ProviderTypeStateDisabled   = "disabled"
)

type ProviderType struct {
	ID       int               `json:"id"`
	TypeName string            `json:"typeName"`
	State    ProviderTypeState `json:"state" enums:"enabled,coming_soon,disabled"`
}

type ProviderTypesResponse []ProviderType
