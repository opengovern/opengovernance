package model

import (
	checkupapi "github.com/kaytu-io/open-governance/pkg/checkup/api"
	"gorm.io/gorm"
)

type CheckupJob struct {
	gorm.Model
	Status         checkupapi.CheckupJobStatus
	FailureMessage string
}
