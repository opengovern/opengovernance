package db

import (
	"time"

	"github.com/jackc/pgtype"
	"gorm.io/gorm"
)


// define task status Enum for the task model
type TaskStatus string

const (
	Running TaskStatus = "running"
	Completed TaskStatus = "completed"
	Failed TaskStatus = "failed"
	
)



type Task struct {
	gorm.Model
	Name          string
	Description   string
	IsCompleted   bool
	CompletedDate time.Time
	LastRunDate   time.Time
	ImageUrl	 string
	Interval	 int
	AutoRun 	bool
	Status 	  TaskStatus 

}

type TaskResult struct {
	gorm.Model
	TaskID        uint
	RunDate       time.Time  
	Result        pgtype.JSONB

}