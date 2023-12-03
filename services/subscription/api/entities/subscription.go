package entities

import "time"

type LifeCycleState = string

const (
	LifeCycleState_PendingFulfillmentStart = "PendingFulfillmentStart"
	LifeCycleState_Subscribed              = "Subscribed"
	LifeCycleState_Suspended               = "Suspended"
	LifeCycleState_Cancelled               = "Cancelled"
)

type Subscription struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time

	LifeCycleState         LifeCycleState
	ProviderSubscriptionID string
	OwnerResolvingToken    string
	OwnerID                *string
}
