package openai

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/sashabaranov/go-openai"
)

//go:embed main_prompt.txt
var mainPromptStr string

type Service struct {
	MainPrompt    string
	Model         string
	AssistantName string
	Tools         []openai.AssistantTool
	Files         map[string]string

	fileIDs []string

	client          *openai.Client
	inventoryClient client.InventoryServiceClient
}

func New(token, baseURL, modelName string, i client.InventoryServiceClient) (*Service, error) {
	config := openai.DefaultAzureConfig(token, baseURL)
	config.APIVersion = "2024-02-15-preview"
	gptClient := openai.NewClientWithConfig(config)

	s := &Service{
		client:          gptClient,
		MainPrompt:      mainPromptStr,
		Model:           modelName,
		AssistantName:   "kaytu-r-assistant",
		inventoryClient: i,
		Tools: []openai.AssistantTool{
			{
				Type: openai.AssistantToolTypeCodeInterpreter,
			},
			{
				Type: openai.AssistantToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        "RunQuery",
					Description: "Run SQL Query on Kaytu",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"query": map[string]any{
								"type": "string", "description": "The SQL Query to run",
							},
						},
						"required": []string{"query"},
					},
				},
			},
		},
	}
	err := s.InitFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to init files due to %v", err)
	}

	err = s.InitAssistant()
	if err != nil {
		return nil, fmt.Errorf("failed to init assistant due to %v", err)
	}

	return s, nil
}

func (s *Service) InitAssistant() error {
	assistants, err := s.client.ListAssistants(context.Background(), nil, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to list assistants due to %v", err)
	}

	var assistant *openai.Assistant
	for _, as := range assistants.Assistants {
		if as.Name != nil && *as.Name == s.AssistantName {
			assistant = &as
		}
	}

	if assistant == nil {
		a, err := s.client.CreateAssistant(context.Background(), openai.AssistantRequest{
			Model:        s.Model,
			Name:         &s.AssistantName,
			Description:  nil,
			Instructions: &s.MainPrompt,
			Tools:        s.Tools,
			FileIDs:      s.fileIDs,
			Metadata:     nil,
		})
		if err != nil {
			return fmt.Errorf("failed to create assistants due to %v", err)
		}
		assistant = &a
	}

	if assistant.Instructions == nil || *assistant.Instructions != s.MainPrompt {
		a, err := s.client.ModifyAssistant(context.Background(), assistant.ID, openai.AssistantRequest{
			Model:        s.Model,
			Name:         &s.AssistantName,
			Description:  nil,
			Instructions: &s.MainPrompt,
			Tools:        s.Tools,
			FileIDs:      s.fileIDs,
			Metadata:     nil,
		})
		if err != nil {
			return fmt.Errorf("failed to modify assistants due to %v", err)
		}
		assistant = &a
	}

	return nil
}

func (s *Service) InitFiles() error {
	files, err := s.client.ListFiles(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list files due to %v", err)
	}

	s.fileIDs = nil
	for filename, content := range s.Files {
		exists := false
		for _, f := range files.Files {
			if f.FileName == filename {
				exists = true
				s.fileIDs = append(s.fileIDs, f.ID)
				break
			}
		}

		if !exists {
			f, err := s.client.CreateFileBytes(context.Background(), openai.FileBytesRequest{
				Name:    filename,
				Bytes:   []byte(content),
				Purpose: "",
			})
			if err != nil {
				return fmt.Errorf("failed to create file due to %v", err)
			}

			s.fileIDs = append(s.fileIDs, f.ID)
		}
	}

	return nil
}
