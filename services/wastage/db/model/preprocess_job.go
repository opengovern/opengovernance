package model

import (
	"gorm.io/gorm"
)

type Status string

const (
	Status_CREATED     Status = "CREATED"
	Status_IN_PROGRESS Status = "IN_PROGRESS"
	Status_SUCCEEDED   Status = "SUCCEEDED"
	Status_FAILED      Status = "FAILED"
)

type PreProcessJob struct {
	gorm.Model

	Auth0UserId string
	Status      Status
}
