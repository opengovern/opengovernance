package model

import (
	"github.com/kaytu-io/kaytu-engine/services/subscription/api/entities"
	"gorm.io/gorm"
)

type Subscription struct {
	gorm.Model

	LifeCycleState         entities.LifeCycleState
	ProviderSubscriptionID string
	OwnerResolvingToken    string
	OwnerID                *string
}
