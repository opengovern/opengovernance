package openai

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	analyticsDB "github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	complianceClient "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/assistant/model"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai/knowledge/builders/controls"
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
	logger *zap.Logger

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

func NewQueryAssistant(logger *zap.Logger, isAzure bool, token, baseURL, modelName, orgId string, c complianceClient.ComplianceServiceClient, prompt repository.Prompt) (*Service, error) {
	var config openai.ClientConfig
	if isAzure {
		config = openai.DefaultAzureConfig(token, baseURL)
		config.APIVersion = "2024-02-15-preview"
	} else {
		config = openai.DefaultConfig(token)
		config.OrgID = orgId
	}
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
		logger:        logger,
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

func NewAssetsAssistant(logger *zap.Logger, isAzure bool, token, baseURL, modelName, orgId string, i inventoryClient.InventoryServiceClient, prompt repository.Prompt) (*Service, error) {
	var config openai.ClientConfig
	if isAzure {
		config = openai.DefaultAzureConfig(token, baseURL)
		config.APIVersion = "2024-02-15-preview"
	} else {
		config = openai.DefaultConfig(token)
		config.OrgID = orgId
	}
	gptClient := openai.NewClientWithConfig(config)

	files := map[string]string{}

	assistantMetrics, err := metrics.ExtractMetrics(logger, i, analyticsDB.MetricTypeAssets)
	if err != nil {
		logger.Error("failed to extract metrics", zap.Error(err))
		return nil, err
	}
	for k, v := range assistantMetrics {
		files[k] = v
	}

	prompts, err := prompt.List(context.Background(), utils.GetPointer(model.AssistantTypeAssets))
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
		logger:         logger,
		client:         gptClient,
		MainPrompt:     mainPrompts,
		ChatPrompt:     chatPrompts,
		Model:          modelName,
		AssistantName:  model.AssistantTypeAssets,
		Files:          files,
		fileIDMap:      make(map[string]string),
		extraVariables: make(map[string]string),
		prompt:         prompt,
		Tools: []openai.AssistantTool{
			{Type: openai.AssistantToolTypeCodeInterpreter},
			{Type: openai.AssistantToolTypeRetrieval},
			{
				Type: openai.AssistantToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        "GetConnectionKaytuIDFromNameOrProviderID",
					Description: "Get connection kaytu id from it's name or provider_id",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name": map[string]any{
								"type":        "string",
								"description": "The name of the connection",
							},
							"provider_id": map[string]any{
								"type":        "string",
								"description": "The id of the connection in the cloud provider",
							},
						},
					},
				},
			},
			{
				Type: openai.AssistantToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        "GetGeneralMetricsTrendsValues",
					Description: "Get general metrics trends values",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"resultLimit": map[string]any{
								"type":        "number",
								"description": "The limit of the result",
							},
							"orderBy": map[string]any{
								"type":        "string",
								"description": "The order by field",
								"enum":        []string{"asc", "dsc"},
							},
							"startDate": map[string]any{
								"type":        "number",
								"description": "The start date in epoch seconds",
							},
							"endDate": map[string]any{
								"type":        "number",
								"description": "The end date in epoch seconds",
							},
							"primaryGoal": map[string]any{
								"type":        "string",
								"description": "The primary goal",
								"enum":        []string{"cloud_account", "metric"},
							},
							"connection": map[string]any{
								"type":        "array",
								"description": "The list of connection ids",
								"items": map[string]any{
									"type": "string",
								},
							},
							"metricId": map[string]any{
								"type":        "array",
								"description": "The list of metric ids",
								"items": map[string]any{
									"type": "string",
								},
							},
						},
						"required": []string{"resultLimit", "order_by", "primary_goal"},
					},
				},
			},
			{
				Type: openai.AssistantToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        "GetDirectionOnMultipleMetricsValues",
					Description: "Get direction on multiple metrics values",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"metricId": map[string]any{
								"type":        "string",
								"description": "The metric id",
							},
							"connections": map[string]any{
								"type":        "array",
								"description": "The list of connection ids",
								"items": map[string]any{
									"type": "string",
								},
							},
							"startDate": map[string]any{
								"type":        "number",
								"description": "The start date in epoch seconds",
							},
							"endDate": map[string]any{
								"type":        "number",
								"description": "The end date in epoch seconds",
							},
						},
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

func NewScoreAssistant(logger *zap.Logger, isAzure bool, token, baseURL, modelName, orgId string, complianceServiceClient complianceClient.ComplianceServiceClient, prompt repository.Prompt) (*Service, error) {
	var config openai.ClientConfig
	if isAzure {
		config = openai.DefaultAzureConfig(token, baseURL)
		config.APIVersion = "2024-02-15-preview"
	} else {
		config = openai.DefaultConfig(token)
		config.OrgID = orgId
	}
	gptClient := openai.NewClientWithConfig(config)

	files := map[string]string{}

	assistantControls, err := controls.ExtractControls(logger, complianceServiceClient, map[string][]string{"score_service_name": nil})
	if err != nil {
		logger.Error("failed to extract metrics", zap.Error(err))
		return nil, err
	}
	for k, v := range assistantControls {
		files[k] = v
	}

	prompts, err := prompt.List(context.Background(), utils.GetPointer(model.AssistantTypeScore))
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
		logger:         logger,
		MainPrompt:     mainPrompts,
		ChatPrompt:     chatPrompts,
		Model:          modelName,
		AssistantName:  model.AssistantTypeScore,
		Files:          files,
		fileIDMap:      map[string]string{},
		extraVariables: map[string]string{},
		client:         gptClient,
		prompt:         prompt,
		Tools: []openai.AssistantTool{
			{Type: openai.AssistantToolTypeCodeInterpreter},
			{Type: openai.AssistantToolTypeRetrieval},
		},
	}

	err = s.InitFiles()
	if err != nil {
		logger.Error("failed to init files", zap.Error(err), zap.String("assistant", string(model.AssistantTypeScore)))
		return nil, fmt.Errorf("failed to init files due to %v", err)
	}

	err = s.InitExtraVariables()
	if err != nil {
		logger.Error("failed to init extra variables", zap.Error(err), zap.String("assistant", string(model.AssistantTypeScore)))
		return nil, fmt.Errorf("failed to init extra variables due to %v", err)
	}

	err = s.InitAssistant()
	if err != nil {
		logger.Error("failed to init assistant", zap.Error(err), zap.String("assistant", string(model.AssistantTypeScore)))
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

	s.logger.Info("main prompt", zap.String("main_prompt", mainPrompt), zap.String("assistant_name", s.AssistantName.String()))

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
			s.logger.Error("failed to create assistants", zap.Error(err), zap.String("assistant_name", s.AssistantName.String()))
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
			s.logger.Error("failed to modify assistants", zap.Error(err), zap.String("assistant_name", s.AssistantName.String()))
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
