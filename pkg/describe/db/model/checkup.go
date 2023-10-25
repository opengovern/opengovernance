package model

import (
	checkupapi "github.com/kaytu-io/kaytu-engine/pkg/checkup/api"
	"gorm.io/gorm"
)

type CheckupJob struct {
	gorm.Model
	Status         checkupapi.CheckupJobStatus
	FailureMessage string
}
