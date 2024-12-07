package model

import (
	"github.com/lib/pq"
	"github.com/opengovern/opencomply/services/describe/api"
	"gorm.io/gorm"
	"strconv"
)

type ComplianceQuickRunStatus string

const (
	ComplianceQuickRunStatusCreated    ComplianceQuickRunStatus = "CREATED"
	ComplianceQuickRunStatusQueued     ComplianceQuickRunStatus = "QUEUED"
	ComplianceQuickRunStatusInProgress ComplianceQuickRunStatus = "IN_PROGRESS"
	ComplianceQuickRunStatusFailed     ComplianceQuickRunStatus = "FAILED"
	ComplianceQuickRunStatusSucceeded  ComplianceQuickRunStatus = "SUCCEEDED"
	ComplianceQuickRunStatusTimeOut    ComplianceQuickRunStatus = "TIMEOUT"
	ComplianceQuickRunStatusCanceled   ComplianceQuickRunStatus = "CANCELED"
)

type ComplianceQuickRun struct {
	gorm.Model
	FrameworkID    string
	IntegrationIDs pq.StringArray `gorm:"type:text[]"`
	IncludeResults pq.StringArray `gorm:"type:text[]"`
	Status         ComplianceQuickRunStatus
	FailureMessage string
	CreatedBy      string
	ParentJobId    *uint

	NatsSequenceNumber uint64
}

func (aj *ComplianceQuickRun) ToAPI() api.ComplianceQuickRun {
	return api.ComplianceQuickRun{
		ID:             strconv.Itoa(int(aj.ID)),
		FrameworkID:    aj.FrameworkID,
		IntegrationIDs: aj.IntegrationIDs,
		Status:         api.ComplianceQuickRunStatus(aj.Status),
		FailureMessage: aj.FailureMessage,
		CreatedBy:      aj.CreatedBy,
	}
}
