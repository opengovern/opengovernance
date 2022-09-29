package types

type FullPolicy struct {
	ID    string
	Title string
}

type PolicyStatus string

const (
	PolicyStatusPASSED  PolicyStatus = "passed"
	PolicyStatusFAILED  PolicyStatus = "failed"
	PolicyStatusUNKNOWN PolicyStatus = "unknown"
)
