package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/assistant/config"
	"github.com/kaytu-io/kaytu-engine/services/assistant/model"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai"
	"github.com/kaytu-io/kaytu-engine/services/assistant/repository"
	openai2 "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"strings"
	"time"
)

type RedirectAssistantActionsService struct {
	logger  *zap.Logger
	oc      *openai.Service
	runRepo repository.Run
	cnf     config.AssistantConfig

	onboardClient onboardClient.OnboardServiceClient
}

func NewRedirectAssistantActions(logger *zap.Logger, cnf config.AssistantConfig, oc *openai.Service, runRepo repository.Run, onboardClient onboardClient.OnboardServiceClient) (Service, error) {
	if oc.AssistantName != model.AssistantTypeRedirection {
		return nil, errors.New(fmt.Sprintf("incompatible assistant type %v", oc.AssistantName))
	}
	return &RedirectAssistantActionsService{
		logger:  logger,
		oc:      oc,
		runRepo: runRepo,
		cnf:     cnf,
	}, nil
}

func (s *RedirectAssistantActionsService) RunActions() {
	for {
		err := s.run()
		if err != nil {
			s.logger.Warn("failed to run due to", zap.Error(err))
		}
		time.Sleep(2 * time.Second)
	}
}

func (s *RedirectAssistantActionsService) run() error {
	runs, err := s.runRepo.List(context.Background(), utils.GetPointer(model.AssistantTypeRedirection))
	if err != nil {
		s.logger.Error("failed to list runs", zap.Error(err))
		return err
	}

	for _, runSummary := range runs {
		if runSummary.Status != openai2.RunStatusRequiresAction &&
			runSummary.Status != openai2.RunStatusQueued &&
			runSummary.Status != openai2.RunStatusInProgress {
			continue
		}

		s.logger.Info("updating run status", zap.String("assistant_type", model.AssistantTypeRedirection.String()), zap.String("run_id", runSummary.ID), zap.String("thread_id", runSummary.ThreadID), zap.String("status", string(runSummary.Status)))

		run, err := s.oc.RetrieveRun(runSummary.ThreadID, runSummary.ID)
		if err != nil {
			return err
		}

		if runSummary.UpdatedAt.Before(time.Now().Add(-15 * time.Second)) {
			err = s.runRepo.UpdateStatus(context.Background(), run.ID, run.ThreadID, run.Status)
			if err != nil {
				return err
			}
		}

		if run.Status == openai2.RunStatusRequiresAction {
			if run.RequiredAction.Type == openai2.RequiredActionTypeSubmitToolOutputs {
				var output []openai2.ToolOutput
				for _, call := range run.RequiredAction.SubmitToolOutputs.ToolCalls {
					if call.Type != openai2.ToolTypeFunction {
						continue
					}
					switch call.Function.Name {
					case "GetFullUrlFromPath":
						out, err := s.GetFullUrlFromPath(call)
						if err != nil {
							s.logger.Error("failed to get full url from path", zap.Error(err))
							out = fmt.Sprintf("Failed to run due to %v", err)
						}
						output = append(output, openai2.ToolOutput{
							ToolCallID: call.ID,
							Output:     out,
						})
					case "GetConnectionKaytuIDFromNameOrProviderID":
						out, err := s.GetConnectionKaytuIDFromNameOrProviderID(call)
						if err != nil {
							s.logger.Error("failed to get connection kaytu id from name or provider id", zap.Error(err))
							out = fmt.Sprintf("Failed to run due to %v", err)
						}
						output = append(output, openai2.ToolOutput{
							ToolCallID: call.ID,
							Output:     out,
						})
					}
				}
				_, err := s.oc.Client().SubmitToolOutputs(
					context.Background(),
					run.ThreadID,
					run.ID,
					openai2.SubmitToolOutputsRequest{
						ToolOutputs: output,
					},
				)
				if err != nil {
					s.logger.Error("failed to submit tool outputs", zap.Error(err))
					return err
				}
			}
		}
	}

	return nil
}

func (s *RedirectAssistantActionsService) GetFullUrlFromPath(call openai2.ToolCall) (string, error) {
	if call.Function.Name != "GetFullUrlFromPath" {
		return "", errors.New(fmt.Sprintf("incompatible function name %v", call.Function.Name))
	}
	var gptArgs map[string]any
	err := json.Unmarshal([]byte(call.Function.Arguments), &gptArgs)
	if err != nil {
		s.logger.Error("failed to unmarshal gpt args", zap.Error(err), zap.String("args", call.Function.Arguments))
		return "", err
	}

	if pathAny, ok := gptArgs["path"]; ok {
		path, ok := pathAny.(string)
		if !ok {
			return "", errors.New(fmt.Sprintf("invalid path %v", pathAny))
		}
		prefix := fmt.Sprintf("https://%s/%s/", s.cnf.KaytuBaseUrl, s.cnf.WorkspaceName)
		if strings.HasPrefix(path, prefix) {
			return path, nil
		}
		path = strings.TrimPrefix(path, "/")
		return fmt.Sprintf("%s%s", prefix, path), nil
	} else {
		return "", errors.New(fmt.Sprintf("path not found in %v", gptArgs))
	}
}

func (s *RedirectAssistantActionsService) GetConnectionKaytuIDFromNameOrProviderID(call openai2.ToolCall) (string, error) {
	if call.Function.Name != "GetConnectionKaytuIDFromNameOrProviderID" {
		return "", errors.New(fmt.Sprintf("incompatible function name %v", call.Function.Name))
	}
	var gptArgs map[string]any
	err := json.Unmarshal([]byte(call.Function.Arguments), &gptArgs)
	if err != nil {
		s.logger.Error("failed to unmarshal gpt args", zap.Error(err), zap.String("args", call.Function.Arguments))
		return "", err
	}

	allConnections, err := s.onboardClient.ListSources(&httpclient.Context{UserRole: authApi.InternalRole}, nil)
	if err != nil {
		s.logger.Error("failed to list sources", zap.Error(err), zap.Any("args", gptArgs))
		return "", fmt.Errorf("there has been a backend error")
	}

	if nameAny, ok := gptArgs["name"]; ok {
		name, ok := nameAny.(string)
		if !ok {
			return "", errors.New(fmt.Sprintf("invalid name %v", nameAny))
		}
		for _, connection := range allConnections {
			if strings.TrimSpace(strings.ToLower(connection.ConnectionName)) == strings.TrimSpace(strings.ToLower(name)) {
				return connection.ID.String(), nil
			}
		}
	}
	if providerIDAny, ok := gptArgs["provider_id"]; ok {
		providerID, ok := providerIDAny.(string)
		if !ok {
			return "", errors.New(fmt.Sprintf("invalid provider_id %v", providerIDAny))
		}
		for _, connection := range allConnections {
			if strings.TrimSpace(strings.ToLower(connection.ConnectionID)) == strings.TrimSpace(strings.ToLower(providerID)) {
				return connection.ID.String(), nil
			}
		}
	}

	s.logger.Error("name or provider_id not found in input", zap.Any("args", gptArgs))
	return "", errors.New(fmt.Sprintf("name or provider_id not found in input"))
}
