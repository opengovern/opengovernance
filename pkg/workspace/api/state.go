package api

type StateID string

const (
	StateID_Reserving     StateID = "RESERVING"
	StateID_Reserved      StateID = "RESERVED"
	StateID_Bootstrapping StateID = "BOOTSTRAPPING"
	StateID_Provisioning  StateID = "PROVISIONING"
	StateID_Provisioned   StateID = "PROVISIONED"
	StateID_Deleting      StateID = "DELETING"
	StateID_Deleted       StateID = "DELETED"
)

func (s StateID) IsReserve() bool {
	return s == StateID_Reserving || s == StateID_Reserved
}
