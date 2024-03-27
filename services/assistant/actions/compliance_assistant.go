package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/goccy/go-yaml"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	complianceClient "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/assistant/config"
	"github.com/kaytu-io/kaytu-engine/services/assistant/model"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai"
	"github.com/kaytu-io/kaytu-engine/services/assistant/repository"
	openai2 "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"net/url"
	"time"
)

type ComplianceAssistantActionsService struct {
	logger  *zap.Logger
	oc      *openai.Service
	runRepo repository.Run
	cnf     config.AssistantConfig

	onboardClient    onboardClient.OnboardServiceClient
	complianceClient complianceClient.ComplianceServiceClient
}

func NewComplianceAssistantActions(logger *zap.Logger, cnf config.AssistantConfig, oc *openai.Service, runRepo repository.Run,
	onboardClient onboardClient.OnboardServiceClient, complianceClient complianceClient.ComplianceServiceClient) (Service, error) {
	if oc.AssistantName != model.AssistantTypeCompliance {
		return nil, errors.New(fmt.Sprintf("incompatible assistant type %v", oc.AssistantName))
	}
	return &ComplianceAssistantActionsService{
		logger:           logger,
		oc:               oc,
		runRepo:          runRepo,
		cnf:              cnf,
		onboardClient:    onboardClient,
		complianceClient: complianceClient,
	}, nil
}

func (s *ComplianceAssistantActionsService) RunActions() {
	for {
		err := s.run()
		if err != nil {
			s.logger.Warn("failed to run due to", zap.Error(err), zap.String("assistant", s.oc.AssistantName.String()))
		}
		time.Sleep(2 * time.Second)
	}
}

func (s *ComplianceAssistantActionsService) run() error {
	runs, err := s.runRepo.List(context.Background(), utils.GetPointer(s.oc.AssistantName))
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

		s.logger.Info("updating run status", zap.String("assistant_type", s.oc.AssistantName.String()), zap.String("run_id", runSummary.ID), zap.String("thread_id", runSummary.ThreadID), zap.String("status", string(runSummary.Status)))

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
					case "GetConnectionKaytuIDFromNameOrProviderID":
						out, err := getConnectionKaytuIDFromNameOrProviderID(s.logger, s.onboardClient, call)
						if err != nil {
							s.logger.Error("failed to get connection kaytu id from name or provider id", zap.Error(err))
							out = fmt.Sprintf("Failed to run due to %v", err)
						}
						output = append(output, openai2.ToolOutput{
							ToolCallID: call.ID,
							Output:     out,
						})
					case "GetDirectionOnBenchmarkResultValues":
						out, err := s.GetDirectionOnBenchmarkResultValues(call)
						if err != nil {
							s.logger.Error("failed to get direction on benchmark result values", zap.Error(err))
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

type GetDirectionOnBenchmarkResultValuesResponse struct {
	Results      []GetDirectionOnBenchmarkResultValuesDataPoint `json:"results" yaml:"results"`
	ReferenceURL string                                         `json:"reference_url" yaml:"reference_url"`
}

type GetDirectionOnBenchmarkResultValuesDataPoint struct {
	Date                          time.Time      `json:"time" yaml:"date"`
	PassedFindingCount            int            `json:"passed_finding_count" yaml:"passed_finding_count"`
	FailedFindingsBySeverityCount map[string]int `json:"failed_findings_by_severity" yaml:"failed_findings_by_severity"`
	PassedControlCount            int            `json:"passed_control_count" yaml:"passed_control_count"`
	FailedControlsBySeverityCount map[string]int `json:"failed_controls_by_severity" yaml:"failed_controls_by_severity"`
}

func (s *ComplianceAssistantActionsService) GetDirectionOnBenchmarkResultValues(call openai2.ToolCall) (string, error) {
	if call.Function.Name != "GetDirectionOnBenchmarkResultValues" {
		return "", errors.New(fmt.Sprintf("incompatible function name %v", call.Function.Name))
	}
	var gptArgs map[string]any
	err := json.Unmarshal([]byte(call.Function.Arguments), &gptArgs)
	if err != nil {
		s.logger.Error("failed to unmarshal gpt args", zap.Error(err), zap.String("args", call.Function.Arguments))
		return "", err
	}

	benchmarkId := ""
	if benchmarkIdAny, ok := gptArgs["benchmarkId"]; ok {
		benchmarkId, ok = benchmarkIdAny.(string)
		if !ok {
			return "", errors.New(fmt.Sprintf("invalid metric_id type %T must be string", benchmarkIdAny))
		}
	} else {
		return "", errors.New(fmt.Sprintf("metric_id not found in %v", gptArgs))
	}

	var startDate *time.Time
	if startDateAny, ok := gptArgs["startDate"]; ok {
		startDateVal, ok := startDateAny.(int64)
		if !ok {
			startDateFloat, ok := startDateAny.(float64)
			if !ok {
				return "", errors.New(fmt.Sprintf("invalid startDate type %T must be int or float", startDateAny))
			}
			startDateInt := int64(startDateFloat)
			startDate = utils.GetPointer(time.Unix(startDateInt, 0))
		} else {
			startDate = utils.GetPointer(time.Unix(startDateVal, 0))
		}
	}

	var endDate *time.Time
	if endDateAny, ok := gptArgs["endDate"]; ok {
		endDateVal, ok := endDateAny.(int64)
		if !ok {
			endDateFloat, ok := endDateAny.(float64)
			if !ok {
				return "", errors.New(fmt.Sprintf("invalid endDate type %T must be int or float", endDateAny))
			}
			endDateInt := int64(endDateFloat)
			endDate = utils.GetPointer(time.Unix(endDateInt, 0))
		} else {
			endDate = utils.GetPointer(time.Unix(endDateVal, 0))
		}
	}

	connections := make([]string, 0)
	if connectionsAny, ok := gptArgs["connections"]; ok {
		connectionsAnyArray, ok := connectionsAny.([]any)
		if !ok {
			return "", errors.New(fmt.Sprintf("invalid connections type %T must be []string", connectionsAnyArray))
		}
		for _, connectionAny := range connectionsAnyArray {
			connection, ok := connectionAny.(string)
			if !ok {
				return "", errors.New(fmt.Sprintf("invalid connection type %T must be string", connectionAny))
			}
			connections = append(connections, connection)
		}
	}

	benchmarkTrendDatapoints, err := s.complianceClient.GetBenchmarkTrend(&httpclient.Context{UserRole: authApi.InternalRole}, benchmarkId, connections, startDate, endDate)
	if err != nil {
		s.logger.Error("failed to get benchmark trend datapoints", zap.Error(err))
		return "", err
	}

	output := GetDirectionOnBenchmarkResultValuesResponse{
		ReferenceURL: fmt.Sprintf("https://%s/%s/compliance/%s", s.cnf.KaytuBaseUrl, s.cnf.WorkspaceName, benchmarkId),
	}
	params := url.Values{}
	if startDate != nil {
		params.Set("startDate", startDate.Format("2006-01-02+15:04:05"))
	}
	if endDate != nil {
		params.Set("endDate", endDate.Format("2006-01-02+15:04:05"))
	}
	if len(connections) > 0 {
		for _, connection := range connections {
			params.Add("connections", connection)
		}
	}
	if len(params) > 0 {
		output.ReferenceURL = fmt.Sprintf("%s?%s", output.ReferenceURL, params.Encode())
	}

	for _, dp := range benchmarkTrendDatapoints {
		adp := GetDirectionOnBenchmarkResultValuesDataPoint{
			Date:                          dp.Timestamp,
			PassedFindingCount:            dp.ConformanceStatusSummary.PassedCount,
			FailedFindingsBySeverityCount: make(map[string]int),
			PassedControlCount:            dp.ControlsSeverityStatus.Total.PassedCount,
			FailedControlsBySeverityCount: make(map[string]int),
		}

		adp.FailedFindingsBySeverityCount["critical"] = dp.Checks.CriticalCount
		adp.FailedFindingsBySeverityCount["high"] = dp.Checks.HighCount
		adp.FailedFindingsBySeverityCount["medium"] = dp.Checks.MediumCount
		adp.FailedFindingsBySeverityCount["low"] = dp.Checks.LowCount
		adp.FailedFindingsBySeverityCount["none"] = dp.Checks.NoneCount

		adp.FailedControlsBySeverityCount["critical"] = dp.ControlsSeverityStatus.Critical.TotalCount
		adp.FailedControlsBySeverityCount["high"] = dp.ControlsSeverityStatus.High.TotalCount
		adp.FailedControlsBySeverityCount["medium"] = dp.ControlsSeverityStatus.Medium.TotalCount
		adp.FailedControlsBySeverityCount["low"] = dp.ControlsSeverityStatus.Low.TotalCount
		adp.FailedControlsBySeverityCount["none"] = dp.ControlsSeverityStatus.None.TotalCount

		output.Results = append(output.Results, adp)
	}

	yamlOutput, err := yaml.Marshal(output)
	if err != nil {
		s.logger.Error("failed to marshal output", zap.Error(err))
		return "", err
	}

	return string(yamlOutput), nil
}
