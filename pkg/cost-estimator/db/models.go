package db

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/gorm"
	"time"
)

type StoreCostTableJobStatus string

const (
	StoreCostTableJobStatusProcessing StoreCostTableJobStatus = "PROCESSING"
	StoreCostTableJobStatusFailed     StoreCostTableJobStatus = "FAILED"
	StoreCostTableJobStatusSucceeded  StoreCostTableJobStatus = "SUCCEEDED"
)

type StoreCostTableJob struct {
	Id           uint `json:"id" sql:"AUTO_INCREMENT" gorm:"primary_key"`
	CreatedAt    time.Time
	UpdatedAt    time.Time      `gorm:"index:,sort:desc"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Connector    source.Type
	ErrorMessage string
	Status       StoreCostTableJobStatus
	Count        int64
}
