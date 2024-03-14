package model

type Purpose string

const (
	Purpose_SystemPrompt = "SystemPrompt"
	Purpose_ChatPrompt   = "ChatPrompt"
)

type Prompt struct {
	Purpose Purpose `gorm:"primarykey"`
	Content string
}
