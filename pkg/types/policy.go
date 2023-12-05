package types

type FullControl struct {
	ID    string `json:"ID" example:"azure_cis_v140_7_5"`                                                            // Control ID
	Title string `json:"title" example:"7.5 Ensure that the latest OS Patches for all Virtual Machines are applied"` // Control title
}

type ControlStatus string

const (
	ControlStatusPASSED  ControlStatus = "passed"
	ControlStatusFAILED  ControlStatus = "failed"
	ControlStatusUNKNOWN ControlStatus = "unknown"
)
