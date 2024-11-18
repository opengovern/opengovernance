package model

import (
	checkupapi "github.com/opengovern/opengovernance/jobs/checkup-job/api"
	"gorm.io/gorm"
)

type CheckupJob struct {
	gorm.Model
	Status         checkupapi.CheckupJobStatus
	FailureMessage string
}
