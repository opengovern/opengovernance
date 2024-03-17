package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/assistant/model"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai"
	"github.com/kaytu-io/kaytu-engine/services/assistant/repository"
	openai2 "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"time"
)

type QueryAssistantActionsService struct {
	logger  *zap.Logger
	oc      *openai.Service
	runRepo repository.Run
	i       client.InventoryServiceClient
}

func NewQueryAssistantActions(logger *zap.Logger, oc *openai.Service, i client.InventoryServiceClient, runRepo repository.Run) (Service, error) {
	if oc.AssistantName != model.AssistantTypeQuery {
		return nil, errors.New(fmt.Sprintf("incompatible assistant type %v", oc.AssistantName))
	}
	return &QueryAssistantActionsService{
		logger:  logger,
		oc:      oc,
		i:       i,
		runRepo: runRepo,
	}, nil
}

func (s *QueryAssistantActionsService) RunActions() {
	for {
		err := s.run()
		if err != nil {
			fmt.Println("failed to run due to", err)
		}
		time.Sleep(2 * time.Second)
	}
}

func (s *QueryAssistantActionsService) run() error {
	runs, err := s.runRepo.List(context.Background(), utils.GetPointer(model.AssistantTypeQuery))
	if err != nil {
		return err
	}

	for _, runSummary := range runs {
		if runSummary.Status != openai2.RunStatusRequiresAction &&
			runSummary.Status != openai2.RunStatusQueued &&
			runSummary.Status != openai2.RunStatusInProgress {
			continue
		}

		s.logger.Info("updating run status", zap.String("assistant_type", model.AssistantTypeQuery.String()), zap.String("run_id", runSummary.ID), zap.String("thread_id", runSummary.ThreadID), zap.String("status", string(runSummary.Status)))

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
					fmt.Printf("run SQL action %v\n", call)
					out, err := s.runSQLQueryAction(call)
					if err != nil {
						out = fmt.Sprintf("Failed to run due to %v", err)
					}
					fmt.Printf("run SQL action out %v\n", out)
					output = append(output, openai2.ToolOutput{
						ToolCallID: call.ID,
						Output:     out,
					})
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
					return err
				}
			}
		}
	}

	return nil
}

func (s *QueryAssistantActionsService) runSQLQueryAction(query openai2.ToolCall) (string, error) {
	if query.Function.Name != "RunQuery" {
		return "", errors.New("invalid function")
	}

	var gptArgs map[string]interface{}
	err := json.Unmarshal([]byte(query.Function.Arguments), &gptArgs)
	if err != nil {
		return "", err
	}

	q, ok := gptArgs["query"].(string)
	if !ok {
		return "", fmt.Errorf("failed to convert query to %v", gptArgs["query"])
	}

	pageNo, ok := gptArgs["pageNo"].(int64)
	if !ok {
		pageNoF, ok := gptArgs["pageNo"].(float64)
		if !ok {
			pageNo = 1
		} else {
			pageNo = int64(pageNoF)
		}
	}

	pageSize, ok := gptArgs["pageSize"].(int64)
	if !ok {
		pageSizeF, ok := gptArgs["pageSize"].(float64)
		if !ok {
			pageSize = 100
		} else {
			pageSize = int64(pageSizeF)
		}
	}

	res, err := s.i.RunQuery(&httpclient.Context{
		UserRole: api2.InternalRole,
	}, api.RunQueryRequest{
		Page:  api.Page{No: int(pageNo), Size: int(pageSize)},
		Query: &q,
		Sorts: nil,
	})
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(res)
	if err != nil {
		return "", err
	}

	return string(out), nil
}
