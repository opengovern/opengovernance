package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	openai2 "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"strings"
)

func getConnectionKaytuIDFromNameOrProviderID(logger *zap.Logger, onboardClient onboardClient.OnboardServiceClient, call openai2.ToolCall) (string, error) {
	if call.Function.Name != "GetConnectionKaytuIDFromNameOrProviderID" {
		return "", errors.New(fmt.Sprintf("incompatible function name %v", call.Function.Name))
	}
	var gptArgs map[string]any
	err := json.Unmarshal([]byte(call.Function.Arguments), &gptArgs)
	if err != nil {
		logger.Error("failed to unmarshal gpt args", zap.Error(err), zap.String("args", call.Function.Arguments))
		return "", err
	}

	allConnections, err := onboardClient.ListSources(&httpclient.Context{UserRole: authApi.InternalRole}, nil)
	if err != nil {
		logger.Error("failed to list sources", zap.Error(err), zap.Any("args", gptArgs))
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

	logger.Error("no connection found", zap.Any("args", gptArgs))
	return "", errors.New(fmt.Sprintf("no connection found for input %v", gptArgs))
}
