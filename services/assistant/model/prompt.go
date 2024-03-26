package model

type Purpose string

const (
	Purpose_SystemPrompt = "SystemPrompt"
	Purpose_ChatPrompt   = "ChatPrompt"
)

type AssistantType string

const (
	AssistantTypeQuery      AssistantType = "kaytu-r-assistant"
	AssistantTypeAssets     AssistantType = "kaytu-assets-assistant"
	AssistantTypeScore      AssistantType = "kaytu-score-assistant"
	AssistantTypeCompliance AssistantType = "kaytu-compliance-assistant"
)

func (a AssistantType) String() string {
	return string(a)
}

type Prompt struct {
	Purpose       Purpose       `gorm:"primaryKey"`
	AssistantName AssistantType `gorm:"primaryKey"`
	Content       string
}
