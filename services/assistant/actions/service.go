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
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai"
	"github.com/kaytu-io/kaytu-engine/services/assistant/repository"
	openai2 "github.com/sashabaranov/go-openai"
	"time"
)

type Service struct {
	oc      *openai.Service
	runRepo repository.Run
	i       client.InventoryServiceClient
}

func New(oc *openai.Service, i client.InventoryServiceClient, runRepo repository.Run) *Service {
	return &Service{
		oc:      oc,
		i:       i,
		runRepo: runRepo,
	}
}

func (s *Service) Run() {
	for {
		err := s.run()
		if err != nil {
			fmt.Println("failed to run due to", err)
		}
		time.Sleep(1 * time.Second)
	}
}

func (s *Service) run() error {
	runs, err := s.runRepo.List(context.Background())
	if err != nil {
		return err
	}

	for _, runSummary := range runs {
		if runSummary.Status != openai2.RunStatusRequiresAction &&
			runSummary.Status != openai2.RunStatusQueued &&
			runSummary.Status != openai2.RunStatusInProgress {
			continue
		}

		fmt.Printf("updating run threadID: %s, runID: %s\n", runSummary.ThreadID, runSummary.ID)

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
					out, err := s.RunSQLQueryAction(call)
					if err != nil {
						out = fmt.Sprintf("Failed to run due to %v", err)
					}
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

func (s *Service) RunSQLQueryAction(query openai2.ToolCall) (string, error) {
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
			return "", fmt.Errorf("failed to convert query to %v", gptArgs["pageNo"])
		}
		pageNo = int64(pageNoF)
	}

	pageSize, ok := gptArgs["pageSize"].(int64)
	if !ok {
		pageSizeF, ok := gptArgs["pageSize"].(float64)
		if !ok {
			return "", fmt.Errorf("failed to convert query to %v", gptArgs["pageSize"])
		}
		pageSize = int64(pageSizeF)
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
