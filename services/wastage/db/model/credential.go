package model

import (
	"gorm.io/gorm"
)

type Credential struct {
	gorm.Model

	Auth0UserId string
	AWSJumpRole string
}
