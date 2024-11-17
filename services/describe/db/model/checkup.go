package model

import (
	checkupapi "github.com/opengovern/opengovernance/jobs/checkup/api"
	"gorm.io/gorm"
)

type CheckupJob struct {
	gorm.Model
	Status         checkupapi.CheckupJobStatus
	FailureMessage string
}
