package model

import "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"

type ConnectionLifecycleState string

const (
	ConnectionLifecycleStateDisabled   ConnectionLifecycleState = "DISABLED"
	ConnectionLifecycleStateDiscovered ConnectionLifecycleState = "DISCOVERED"
	ConnectionLifecycleStateInProgress ConnectionLifecycleState = "IN_PROGRESS"
	ConnectionLifecycleStateOnboard    ConnectionLifecycleState = "ONBOARD"
	ConnectionLifecycleStateArchived   ConnectionLifecycleState = "ARCHIVED"
)

func (c ConnectionLifecycleState) IsEnabled() bool {
	for _, state := range GetConnectionLifecycleStateEnabledStates() {
		if c == state {
			return true
		}
	}
	return false
}

func GetConnectionLifecycleStateEnabledStates() []ConnectionLifecycleState {
	return []ConnectionLifecycleState{ConnectionLifecycleStateOnboard, ConnectionLifecycleStateInProgress}
}

func (c ConnectionLifecycleState) ToApi() api.ConnectionLifecycleState {
	return api.ConnectionLifecycleState(c)
}

func ConnectionLifecycleStateFromApi(state api.ConnectionLifecycleState) ConnectionLifecycleState {
	return ConnectionLifecycleState(state)
}
