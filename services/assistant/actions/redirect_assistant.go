package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/goccy/go-yaml"
	analyticsDB "github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/assistant/config"
	"github.com/kaytu-io/kaytu-engine/services/assistant/model"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai"
	"github.com/kaytu-io/kaytu-engine/services/assistant/repository"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	openai2 "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"sort"
	"strings"
	"time"
)

type RedirectAssistantActionsService struct {
	logger  *zap.Logger
	oc      *openai.Service
	runRepo repository.Run
	cnf     config.AssistantConfig

	onboardClient   onboardClient.OnboardServiceClient
	inventoryClient inventoryClient.InventoryServiceClient
}

func NewRedirectAssistantActions(logger *zap.Logger, cnf config.AssistantConfig, oc *openai.Service, runRepo repository.Run,
	onboardClient onboardClient.OnboardServiceClient, inventoryClient inventoryClient.InventoryServiceClient) (Service, error) {
	if oc.AssistantName != model.AssistantTypeRedirection {
		return nil, errors.New(fmt.Sprintf("incompatible assistant type %v", oc.AssistantName))
	}
	return &RedirectAssistantActionsService{
		logger:          logger,
		oc:              oc,
		runRepo:         runRepo,
		cnf:             cnf,
		onboardClient:   onboardClient,
		inventoryClient: inventoryClient,
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
					case "GetMetricValues":
						out, err := s.GetMetricValues(call)
						if err != nil {
							s.logger.Error("failed to get metric values", zap.Error(err))
							out = fmt.Sprintf("Failed to run due to %v", err)
						}
						output = append(output, openai2.ToolOutput{
							ToolCallID: call.ID,
							Output:     out,
						})
					case "GetGeneralMetricsTrendsValues":
						out, err := s.GetGeneralMetricsTrendsValues(call)
						if err != nil {
							s.logger.Error("failed to get general metrics trends values", zap.Error(err))
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

type AssistantTrendDataPoint struct {
	Value float64   `json:"value" yaml:"value"`
	Date  time.Time `json:"time" yaml:"date"`
}

func (s *RedirectAssistantActionsService) GetMetricValues(call openai2.ToolCall) (string, error) {
	if call.Function.Name != "GetMetricValues" {
		return "", errors.New(fmt.Sprintf("incompatible function name %v", call.Function.Name))
	}
	var gptArgs map[string]any
	err := json.Unmarshal([]byte(call.Function.Arguments), &gptArgs)
	if err != nil {
		s.logger.Error("failed to unmarshal gpt args", zap.Error(err), zap.String("args", call.Function.Arguments))
		return "", err
	}

	metricType := ""
	if metricTypeAny, ok := gptArgs["metric_type"]; ok {
		metricType, ok = metricTypeAny.(string)
		if !ok {
			return "", errors.New(fmt.Sprintf("invalid metric_type %v", metricTypeAny))
		}
		metricType = strings.ToLower(metricType)
		if metricType != string(analyticsDB.MetricTypeAssets) && metricType != string(analyticsDB.MetricTypeSpend) {
			return "", errors.New(fmt.Sprintf("invalid metric_type %v must be %s or %s", metricType, analyticsDB.MetricTypeAssets, analyticsDB.MetricTypeSpend))
		}
	} else {
		return "", errors.New(fmt.Sprintf("metric_type not found in %v", gptArgs))
	}

	metricId := ""
	if metricIdAny, ok := gptArgs["metric_id"]; ok {
		metricId, ok = metricIdAny.(string)
		if !ok {
			return "", errors.New(fmt.Sprintf("invalid metric_id type %T must be string", metricIdAny))
		}
	} else {
		return "", errors.New(fmt.Sprintf("metric_id not found in %v", gptArgs))
	}

	startTime := int64(0)
	if startTimeAny, ok := gptArgs["start_time"]; ok {
		startTime, ok = startTimeAny.(int64)
		if !ok {
			startTimeFloat, ok := startTimeAny.(float64)
			if !ok {
				return "", errors.New(fmt.Sprintf("invalid start_time type %T must be int or float", startTimeAny))
			}
			startTime = int64(startTimeFloat)
		}
	} else {
		return "", errors.New(fmt.Sprintf("start_time not found in %v", gptArgs))
	}
	endTime := int64(0)
	if endTimeAny, ok := gptArgs["end_time"]; ok {
		endTime, ok = endTimeAny.(int64)
		if !ok {
			endTimeFloat, ok := endTimeAny.(float64)
			if !ok {
				return "", errors.New(fmt.Sprintf("invalid end_time type %T must be int or float", endTimeAny))
			}
			endTime = int64(endTimeFloat)
		}
	} else {
		return "", errors.New(fmt.Sprintf("end_time not found in %v", gptArgs))
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

	result := make([]AssistantTrendDataPoint, 0)

	switch metricType {
	case string(analyticsDB.MetricTypeAssets):
		trendDatapoints, err := s.inventoryClient.ListAnalyticsMetricTrend(&httpclient.Context{UserRole: authApi.InternalRole},
			[]string{metricId}, connections,
			utils.GetPointer(time.Unix(startTime, 0)),
			utils.GetPointer(time.Unix(endTime, 0)))
		if err != nil {
			s.logger.Error("failed to list analytics metric trend", zap.Error(err))
			return "", fmt.Errorf("there has been a backend error: %v", err)
		}
		for _, trendDatapoint := range trendDatapoints {
			result = append(result, AssistantTrendDataPoint{
				Value: float64(trendDatapoint.Count),
				Date:  trendDatapoint.Date,
			})

		}
	case string(analyticsDB.MetricTypeSpend):
		trendDatapoints, err := s.inventoryClient.ListAnalyticsMetricTrend(&httpclient.Context{UserRole: authApi.InternalRole},
			[]string{metricId}, connections,
			utils.GetPointer(time.Unix(startTime, 0)),
			utils.GetPointer(time.Unix(endTime, 0)))
		if err != nil {
			s.logger.Error("failed to list analytics metric trend", zap.Error(err))
			return "", fmt.Errorf("there has been a backend error: %v", err)
		}
		for _, trendDatapoint := range trendDatapoints {
			result = append(result, AssistantTrendDataPoint{
				Value: float64(trendDatapoint.Count),
				Date:  trendDatapoint.Date,
			})
		}
	}

	//sort
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})

	resultYaml, err := yaml.Marshal(result)
	if err != nil {
		s.logger.Error("failed to marshal result", zap.Error(err))
		return "", fmt.Errorf("failed to marshal result: %v", err)
	}

	return string(resultYaml), nil
}

type AssistantGeneralMetricsConnectionData struct {
	KaytuConnectionID      string      `json:"kaytu_connection_id" yaml:"kaytu_connection_id"`
	ProviderConnectionID   string      `json:"provider_connection_id" yaml:"provider_connection_id"`
	ProviderConnectionName string      `json:"provider_connection_name" yaml:"provider_connection_name"`
	Provider               source.Type `json:"provider" yaml:"provider"`
	ResourceCount          *int        `json:"resource_count" yaml:"resource_count"`
}

type AssistantGeneralMetricsMetricData struct {
	MetricID      string `json:"metric_id" yaml:"metric_id"`
	MetricName    string `json:"metric_name" yaml:"metric_name"`
	ResourceCount *int   `json:"resource_count" yaml:"resource_count"`
}

func (s *RedirectAssistantActionsService) GetGeneralMetricsTrendsValues(call openai2.ToolCall) (string, error) {
	if call.Function.Name != "GetGeneralMetricsTrendsValues" {
		return "", errors.New(fmt.Sprintf("incompatible function name %v", call.Function.Name))
	}
	var gptArgs map[string]any
	err := json.Unmarshal([]byte(call.Function.Arguments), &gptArgs)
	if err != nil {
		s.logger.Error("failed to unmarshal gpt args", zap.Error(err), zap.String("args", call.Function.Arguments))
		return "", err
	}

	resultLimit := int64(0)
	if resultLimitAny, ok := gptArgs["resultLimit"]; ok {
		resultLimit, ok = resultLimitAny.(int64)
		if !ok {
			resultLimitFloat, ok := resultLimitAny.(float64)
			if !ok {
				return "", errors.New(fmt.Sprintf("invalid resultLimit type %T must be int or float", resultLimitAny))
			}
			resultLimit = int64(resultLimitFloat)
		}
	} else {
		return "", errors.New(fmt.Sprintf("resultLimit not found in %v", gptArgs))
	}

	orderBy := ""
	if orderByAny, ok := gptArgs["orderBy"]; ok {
		orderBy, ok = orderByAny.(string)
		if !ok {
			return "", errors.New(fmt.Sprintf("invalid orderBy type %T must be string", orderByAny))
		}
		if orderBy != "asc" && orderBy != "dsc" {
			return "", errors.New(fmt.Sprintf("invalid orderBy %v must be asc or dsc", orderBy))
		}
	} else {
		return "", errors.New(fmt.Sprintf("orderBy not found in %v", gptArgs))
	}

	primaryGoal := ""
	if primaryGoalAny, ok := gptArgs["primaryGoal"]; ok {
		primaryGoal, ok = primaryGoalAny.(string)
		if !ok {
			return "", errors.New(fmt.Sprintf("invalid primaryGoal type %T must be string", primaryGoalAny))
		}
		if primaryGoal != "cloud_account" && primaryGoal != "metric" {
			return "", errors.New(fmt.Sprintf("invalid primaryGoal %v must be cloud_account or metric", primaryGoal))
		}
	} else {
		return "", errors.New(fmt.Sprintf("primaryGoal not found in %v", gptArgs))
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

	metricIds := make([]string, 0)
	if metricIdsAny, ok := gptArgs["metricId"]; ok {
		metricIdsAnyArray, ok := metricIdsAny.([]any)
		if !ok {
			return "", errors.New(fmt.Sprintf("invalid metric_id type %T must be []string", metricIdsAny))
		}
		for _, metricIdAny := range metricIdsAnyArray {
			metricId, ok := metricIdAny.(string)
			if !ok {
				return "", errors.New(fmt.Sprintf("invalid metric_id type %T must be string", metricIdAny))
			}
			metricIds = append(metricIds, metricId)
		}
	}

	switch primaryGoal {
	case "cloud_account":
		connectionsData, err := s.inventoryClient.ListConnectionsData(&httpclient.Context{UserRole: authApi.InternalRole}, connections, nil,
			startDate, endDate, metricIds, false, true)
		if err != nil {
			s.logger.Error("failed to list connections data", zap.Error(err))
			return "", fmt.Errorf("there has been a backend error: %v", err)
		}
		connectionsDataArr := make([]inventoryApi.ConnectionData, 0, len(connectionsData))
		for _, connectionData := range connectionsData {
			connectionsDataArr = append(connectionsDataArr, connectionData)
		}
		sort.Slice(connectionsDataArr, func(i, j int) bool {
			switch orderBy {
			case "asc":
				if connectionsDataArr[i].Count == nil {
					return false
				}
				if connectionsDataArr[j].Count == nil {
					return true
				}
				return *connectionsDataArr[i].Count < *connectionsDataArr[j].Count
			case "dsc":
				if connectionsDataArr[i].Count == nil {
					return false
				}
				if connectionsDataArr[j].Count == nil {
					return true
				}
				return *connectionsDataArr[i].Count > *connectionsDataArr[j].Count
			}
			return false
		})
		connectionsDataArr = connectionsDataArr[:min(int(resultLimit), len(connectionsDataArr))]

		connectionIds := make([]string, 0, len(connectionsDataArr))
		for _, connectionData := range connectionsDataArr {
			connectionIds = append(connectionIds, connectionData.ConnectionID)
		}

		connectionsMetadata, err := s.onboardClient.GetSources(&httpclient.Context{UserRole: authApi.InternalRole}, connectionIds)
		if err != nil {
			s.logger.Error("failed to get sources", zap.Error(err))
			return "", fmt.Errorf("there has been a backend error: %v", err)
		}
		connectionsMetadataMap := make(map[string]onboardApi.Connection, len(connectionsMetadata))
		for _, connectionMetadata := range connectionsMetadata {
			connectionsMetadataMap[connectionMetadata.ID.String()] = connectionMetadata
		}

		result := make([]AssistantGeneralMetricsConnectionData, 0, len(connectionsDataArr))
		for _, connectionData := range connectionsDataArr {
			result = append(result, AssistantGeneralMetricsConnectionData{
				KaytuConnectionID:      connectionData.ConnectionID,
				ProviderConnectionID:   connectionsMetadataMap[connectionData.ConnectionID].ConnectionID,
				ProviderConnectionName: connectionsMetadataMap[connectionData.ConnectionID].ConnectionName,
				Provider:               connectionsMetadataMap[connectionData.ConnectionID].Connector,
				ResourceCount:          connectionData.Count,
			})
		}

		resultYaml, err := yaml.Marshal(result)
		if err != nil {
			s.logger.Error("failed to marshal result", zap.Error(err))
			return "", fmt.Errorf("failed to marshal result: %v", err)
		}

		return string(resultYaml), nil
	case "metric":
		metricResponse, err := s.inventoryClient.ListAnalyticsMetricsSummary(&httpclient.Context{UserRole: authApi.InternalRole},
			utils.GetPointer(analyticsDB.MetricTypeAssets), metricIds, connections, startDate, endDate)
		if err != nil {
			s.logger.Error("failed to list analytics metrics summary", zap.Error(err))
			return "", fmt.Errorf("there has been a backend error: %v", err)
		}

		metricsData := make([]inventoryApi.Metric, 0, len(metricResponse.Metrics))
		for _, metric := range metricResponse.Metrics {
			metricsData = append(metricsData, metric)
		}
		sort.Slice(metricsData, func(i, j int) bool {
			switch orderBy {
			case "asc":
				if metricsData[i].Count == nil {
					return false
				}
				if metricsData[j].Count == nil {
					return true
				}
				return *metricsData[i].Count < *metricsData[j].Count
			case "dsc":
				if metricsData[i].Count == nil {
					return false
				}
				if metricsData[j].Count == nil {
					return true
				}
				return *metricsData[i].Count > *metricsData[j].Count
			}
			return false
		})

		metricsData = metricsData[:min(int(resultLimit), len(metricsData))]
		result := make([]AssistantGeneralMetricsMetricData, 0, len(metricsData))

		for _, metricData := range metricsData {
			result = append(result, AssistantGeneralMetricsMetricData{
				MetricID:      metricData.ID,
				MetricName:    metricData.Name,
				ResourceCount: metricData.Count,
			})
		}

		resultYaml, err := yaml.Marshal(result)
		if err != nil {
			s.logger.Error("failed to marshal result", zap.Error(err))
			return "", fmt.Errorf("failed to marshal result: %v", err)
		}

		return string(resultYaml), nil
	}

	return "", errors.New(fmt.Sprintf("invalid primaryGoal %v", primaryGoal))
}
