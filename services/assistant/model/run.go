package model

import (
	openai2 "github.com/sashabaranov/go-openai"
	"time"
)

type Run struct {
	ID            string `gorm:"primarykey"`
	ThreadID      string `gorm:"primarykey"`
	AssistantType AssistantType
	Status        openai2.RunStatus
	UpdatedAt     time.Time
}
