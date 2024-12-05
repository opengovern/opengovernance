package model

import (
	"github.com/lib/pq"
	"github.com/opengovern/opencomply/services/describe/api"
	"gorm.io/gorm"
	"strconv"
)

type AuditJobStatus string

const (
	AuditJobStatusCreated    AuditJobStatus = "CREATED"
	AuditJobStatusQueued     AuditJobStatus = "QUEUED"
	AuditJobStatusInProgress AuditJobStatus = "IN_PROGRESS"
	AuditJobStatusFailed     AuditJobStatus = "FAILED"
	AuditJobStatusSucceeded  AuditJobStatus = "SUCCEEDED"
	AuditJobStatusTimeOut    AuditJobStatus = "TIMEOUT"
	AuditJobStatusCanceled   AuditJobStatus = "CANCELED"
)

type AuditJob struct {
	gorm.Model
	FrameworkID    string
	IntegrationIDs pq.StringArray `gorm:"type:text[]"`
	Status         AuditJobStatus
	FailureMessage string
	CreatedBy      string

	NatsSequenceNumber uint64
}

func (aj *AuditJob) ToAPI() api.AuditJob {
	return api.AuditJob{
		ID:             strconv.Itoa(int(aj.ID)),
		FrameworkID:    aj.FrameworkID,
		IntegrationIDs: aj.IntegrationIDs,
		Status:         api.AuditJobStatus(aj.Status),
		FailureMessage: aj.FailureMessage,
		CreatedBy:      aj.CreatedBy,
	}
}
