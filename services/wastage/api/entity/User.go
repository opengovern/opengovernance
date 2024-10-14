package entity

import (
	"github.com/opengovern/opengovernance/services/wastage/db/model"
	"time"
)

type User struct {
	UserId       string     `json:"user_id"`
	PremiumUntil *time.Time `json:"premium_until"`
}

// ToModel convert to model.User
func (u *User) ToModel() *model.User {
	return &model.User{
		UserId:       u.UserId,
		PremiumUntil: u.PremiumUntil,
	}
}
