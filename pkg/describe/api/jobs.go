package api

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

type GetCredsForJobRequest struct {
	SourceID string `json:"sourceId"`
}

type GetCredsForJobResponse struct {
	Credentials string `json:"creds"`
}

type DescribeJob struct {
	JobID         uint // DescribeResourceJob ID
	ScheduleJobID uint
	ParentJobID   uint // DescribeSourceJob ID
	ResourceType  string
	SourceID      string
	AccountID     string
	DescribedAt   int64
	SourceType    source.Type
	ConfigReg     string
	TriggerType   enums.DescribeTriggerType
}

type DescribeJobResult struct {
	JobID       uint
	ParentJobID uint
	Status      DescribeResourceJobStatus
	Error       string
	DescribeJob DescribeJob
}

type DescribeConnectionJobResult struct {
	JobID  uint
	Result map[uint]DescribeJobResult // DescribeResourceJob ID -> DescribeJobResult
}

type JobCallbackRequest struct {
	SourceID  string                      `json:"sourceId" validate:"required"`
	BlobName  string                      `json:"blobName" validate:"required"`
	JobResult DescribeConnectionJobResult `json:"jobResult" validate:"required"`
}
