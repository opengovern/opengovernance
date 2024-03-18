package openai

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	client4 "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/assistant/model"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai/knowledge/builders/examples"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai/knowledge/builders/jsonmodels"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai/knowledge/builders/metrics"
	tables2 "github.com/kaytu-io/kaytu-engine/services/assistant/openai/knowledge/builders/tables"
	"github.com/kaytu-io/kaytu-engine/services/assistant/repository"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"text/template"
	"time"
)

type Service struct {
	MainPrompt    string
	ChatPrompt    string
	Model         string
	AssistantName model.AssistantType
	Tools         []openai.AssistantTool
	Files         map[string]string

	fileIDs   []string
	fileIDMap map[string]string

	extraVariables map[string]string

	client    *openai.Client
	assistant *openai.Assistant
	prompt    repository.Prompt
}

func NewQueryAssistant(logger *zap.Logger, token, baseURL, modelName string, c client4.ComplianceServiceClient, prompt repository.Prompt) (*Service, error) {
	config := openai.DefaultAzureConfig(token, baseURL)
	config.APIVersion = "2024-02-15-preview"
	gptClient := openai.NewClientWithConfig(config)

	files := map[string]string{}

	for k, v := range jsonmodels.ExtractJSONModels() {
		files[k] = v
	}

	tf, err := tables2.ExtractTableFiles(logger)
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

	prompts, err := prompt.List(context.Background(), utils.GetPointer(model.AssistantTypeQuery))
	if err != nil {
		return nil, err
	}

	var mainPrompts, chatPrompts string
	for _, p := range prompts {
		if p.Purpose == model.Purpose_SystemPrompt {
			mainPrompts = p.Content
		}
		if p.Purpose == model.Purpose_ChatPrompt {
			chatPrompts = p.Content
		}
	}
	s := &Service{
		client:        gptClient,
		MainPrompt:    mainPrompts,
		ChatPrompt:    chatPrompts,
		Model:         modelName,
		AssistantName: model.AssistantTypeQuery,
		Files:         files,
		fileIDMap:     map[string]string{},
		prompt:        prompt,
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

func NewRedirectionAssistant(logger *zap.Logger, token, baseURL, modelName string, i inventoryClient.InventoryServiceClient, prompt repository.Prompt) (*Service, error) {
	config := openai.DefaultAzureConfig(token, baseURL)
	config.APIVersion = "2024-02-15-preview"
	gptClient := openai.NewClientWithConfig(config)

	files := map[string]string{}

	assistantMetrics, err := metrics.ExtractMetrics(logger, i)
	if err != nil {
		logger.Error("failed to extract metrics", zap.Error(err))
		return nil, err
	}
	for k, v := range assistantMetrics {
		files[k] = v
	}

	prompts, err := prompt.List(context.Background(), utils.GetPointer(model.AssistantTypeRedirection))
	if err != nil {
		logger.Error("failed to list prompts", zap.Error(err))
		return nil, err
	}

	var mainPrompts, chatPrompts string
	for _, p := range prompts {
		if p.Purpose == model.Purpose_SystemPrompt {
			mainPrompts = p.Content
		}
		if p.Purpose == model.Purpose_ChatPrompt {
			chatPrompts = p.Content
		}
	}
	s := &Service{
		client:         gptClient,
		MainPrompt:     mainPrompts,
		ChatPrompt:     chatPrompts,
		Model:          modelName,
		AssistantName:  model.AssistantTypeRedirection,
		Files:          files,
		fileIDMap:      make(map[string]string),
		extraVariables: make(map[string]string),
		prompt:         prompt,
		Tools: []openai.AssistantTool{
			{Type: openai.AssistantToolTypeCodeInterpreter},
			{
				Type: openai.AssistantToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        "GetFullUrlFromPath",
					Description: "Get full URL from provided path",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"path": map[string]any{
								"type":        "string",
								"description": "The path to get full URL",
							},
						},
						"required": []string{"path"},
					},
				},
			},
		},
	}

	err = s.InitFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to init files due to %v", err)
	}

	err = s.InitExtraVariables()
	if err != nil {
		return nil, fmt.Errorf("failed to init extra variables due to %v", err)
	}

	err = s.InitAssistant()
	if err != nil {
		return nil, fmt.Errorf("failed to init assistant due to %v", err)
	}

	return s, nil
}

func (s *Service) GetFileID(filename string) string {
	return s.fileIDMap[filename]
}

func (s *Service) GetExtraVariable(variable string) string {
	return s.extraVariables[variable]
}

func (s *Service) InitAssistant() error {
	tmpl := template.New("test")
	tm, err := tmpl.Parse(s.MainPrompt)
	if err != nil {
		panic(err)
	}
	var outputExecute bytes.Buffer
	err = tm.Execute(&outputExecute, s)
	if err != nil {
		panic(err)
	}
	mainPrompt := outputExecute.String()

	fmt.Println("main prompt:", mainPrompt)

	assistants, err := s.client.ListAssistants(context.Background(), nil, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to list assistants due to %v", err)
	}

	var assistant *openai.Assistant
	for _, as := range assistants.Assistants {
		if as.Name != nil && *as.Name == s.AssistantName.String() {
			assistant = &as
		}
	}

	if assistant == nil {
		a, err := s.client.CreateAssistant(context.Background(), openai.AssistantRequest{
			Model:        s.Model,
			Name:         utils.GetPointer(s.AssistantName.String()),
			Description:  nil,
			Instructions: &mainPrompt,
			Tools:        s.Tools,
			FileIDs:      s.fileIDs,
			Metadata:     nil,
		})
		if err != nil {
			return fmt.Errorf("failed to create assistants due to %v", err)
		}
		assistant = &a
	}

	updateFiles := false
	for _, tf := range s.fileIDs {

		exists := false
		for _, fid := range assistant.FileIDs {
			if fid == tf {
				exists = true
			}
		}

		if !exists {
			updateFiles = true
		}
	}

	if updateFiles || assistant.Instructions == nil || *assistant.Instructions != mainPrompt {
		a, err := s.client.ModifyAssistant(context.Background(), assistant.ID, openai.AssistantRequest{
			Model:        s.Model,
			Name:         utils.GetPointer(s.AssistantName.String()),
			Description:  nil,
			Instructions: &mainPrompt,
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
				s.fileIDMap[filename] = f.ID
				break
			}
		}

		if !exists {
			f, err := s.client.CreateFileBytes(context.Background(), openai.FileBytesRequest{
				Name:    filename,
				Bytes:   []byte(content),
				Purpose: openai.PurposeAssistants,
			})
			if err != nil {
				return fmt.Errorf("failed to create file due to %v", err)
			}

			s.fileIDs = append(s.fileIDs, f.ID)
			s.fileIDMap[filename] = f.ID
		}
	}

	return nil
}

func (s *Service) InitExtraVariables() error {
	s.extraVariables["currentDate"] = time.Now().Format("2006-01-02")
	return nil
}

func (s *Service) Client() *openai.Client {
	return s.client
}
