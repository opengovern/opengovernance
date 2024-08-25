package model

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
