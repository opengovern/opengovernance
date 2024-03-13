package model

import (
	openai2 "github.com/sashabaranov/go-openai"
	"time"
)

type Run struct {
	ID        string
	ThreadID  string
	Status    openai2.RunStatus
	UpdatedAt time.Time
}
