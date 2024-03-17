package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/assistant/config"
	"github.com/kaytu-io/kaytu-engine/services/assistant/model"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai"
	"github.com/kaytu-io/kaytu-engine/services/assistant/repository"
	openai2 "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"time"
)

type RedirectAssistantActionsService struct {
	logger  *zap.Logger
	oc      *openai.Service
	runRepo repository.Run
	cnf     config.AssistantConfig
}

func NewRedirectAssistantActions(logger *zap.Logger, cnf config.AssistantConfig, oc *openai.Service, runRepo repository.Run) (Service, error) {
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
					if call.Function.Name == "GetFullUrlFromPath" {
						out, err := s.GetFullUrlFromPath(call)
						if err != nil {
							s.logger.Error("failed to get full url from path", zap.Error(err))
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
		return "", err
	}

	if pathAny, ok := gptArgs["path"]; ok {
		path, ok := pathAny.(string)
		if !ok {
			return "", errors.New(fmt.Sprintf("invalid path %v", pathAny))
		}
		return fmt.Sprintf("%s/%s/%s", s.cnf.KaytuBaseUrl, s.cnf.WorkspaceName, path), nil
	} else {
		return "", errors.New(fmt.Sprintf("path not found in %v", gptArgs))
	}
}
