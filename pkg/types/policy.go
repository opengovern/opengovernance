package types

type FullPolicy struct {
	ID    string `json:"ID" example:"azure_cis_v140_7_5"`                                                            // Policy ID
	Title string `json:"title" example:"7.5 Ensure that the latest OS Patches for all Virtual Machines are applied"` // Policy title
}

type PolicyStatus string

const (
	PolicyStatusPASSED  PolicyStatus = "passed"
	PolicyStatusFAILED  PolicyStatus = "failed"
	PolicyStatusUNKNOWN PolicyStatus = "unknown"
)
