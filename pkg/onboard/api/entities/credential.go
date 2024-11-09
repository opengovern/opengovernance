package entities

import (
	"github.com/opengovern/opengovernance/pkg/onboard/api"
)

func NewCredential(credential any) api.Credential {
	apiCredential := api.Credential{

		Config: "",

		Connections:           nil,
		TotalConnections:      nil,
		OnboardConnections:    nil,
		UnhealthyConnections:  nil,
		DiscoveredConnections: nil,
	}

	return apiCredential
}
