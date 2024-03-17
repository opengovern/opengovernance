package model

type Purpose string

const (
	Purpose_SystemPrompt = "SystemPrompt"
	Purpose_ChatPrompt   = "ChatPrompt"
)

type AssistantType string

const (
	AssistantTypeQuery       = "kaytu-r-assistant"
	AssistantTypeRedirection = "kaytu-redirection-assistant"
)

func (a AssistantType) String() string {
	return string(a)
}

type Prompt struct {
	Purpose       Purpose       `gorm:"primarykey"`
	AssistantName AssistantType `gorm:"primarykey"`
	Content       string
}
