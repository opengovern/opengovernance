package model

type Purpose string

const (
	Purpose_SystemPrompt = "SystemPrompt"
	Purpose_ChatPrompt   = "ChatPrompt"
)

type AssistantType string

const (
	AssistantTypeQuery       AssistantType = "kaytu-r-assistant"
	AssistantTypeRedirection AssistantType = "kaytu-redirection-assistant"
)

func (a AssistantType) String() string {
	return string(a)
}

type Prompt struct {
	Purpose       Purpose       `gorm:"primaryKey"`
	AssistantName AssistantType `gorm:"primaryKey"`
	Content       string
}
