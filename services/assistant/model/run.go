package model

import (
	openai2 "github.com/sashabaranov/go-openai"
	"time"
)

type Run struct {
	ID        string `gorm:"primarykey"`
	ThreadID  string `gorm:"primarykey"`
	Status    openai2.RunStatus
	UpdatedAt time.Time
}
