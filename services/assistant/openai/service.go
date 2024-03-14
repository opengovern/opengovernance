package openai

import (
	"context"
	_ "embed"
	"fmt"
	client4 "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai/knowledge/builders/examples"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai/knowledge/builders/jsonmodels"
	tables2 "github.com/kaytu-io/kaytu-engine/services/assistant/openai/knowledge/builders/tables"
	"github.com/sashabaranov/go-openai"
)

//go:embed main_prompt.txt
var mainPromptStr string

//go:embed chat_prompt.txt
var chatPromptStr string

type Service struct {
	MainPrompt    string
	Model         string
	AssistantName string
	Tools         []openai.AssistantTool
	Files         map[string]string

	fileIDs []string

	client          *openai.Client
	inventoryClient client.InventoryServiceClient
	assistant       *openai.Assistant
}

func New(token, baseURL, modelName string, i client.InventoryServiceClient, c client4.ComplianceServiceClient) (*Service, error) {
	config := openai.DefaultAzureConfig(token, baseURL)
	config.APIVersion = "2024-02-15-preview"
	gptClient := openai.NewClientWithConfig(config)

	var files map[string]string
	for k, v := range jsonmodels.ExtractJSONModels() {
		files[k] = v
	}

	tf, err := tables2.ExtractTableFiles()
	if err != nil {
		return nil, err
	}
	for k, v := range tf {
		files[k] = v
	}

	ex, err := examples.ExtractExamples(c)
	if err != nil {
		return nil, err
	}
	for k, v := range ex {
		files[k] = v
	}

	s := &Service{
		client:          gptClient,
		MainPrompt:      mainPromptStr,
		Model:           modelName,
		AssistantName:   "kaytu-r-assistant",
		inventoryClient: i,
		Files:           files,
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
							"pageNo": map[string]any{
								"type": "number", "description": "Result page number starting from 1",
							},
							"pageSize": map[string]any{
								"type": "number", "description": "Result page size, must be a non-zero number",
							},
						},
						"required": []string{"query"},
					},
				},
			},
		},
	}

	err = s.InitFiles()
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

	s.assistant = assistant
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

func (s *Service) Client() *openai.Client {
	return s.client
}
