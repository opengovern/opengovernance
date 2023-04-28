package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"go.uber.org/zap"
)

type LambdaDescribeWorkerInput struct {
	WorkspaceId      string `json:"workspaceId"`
	ConnectionId     string `json:"connectionId"`
	ResourceType     string `json:"resourceType"`
	DescribeEndpoint string `json:"describeEndpoint"`
	KeyARN           string `json:"keyARN"`
}

func DescribeHandler(ctx context.Context, req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	logger.Info(req.Body)

	var input LambdaDescribeWorkerInput
	err = json.Unmarshal([]byte(req.Body), &input)
	if err != nil {
		logger.Error("Failed to unmarshal input", zap.Error(err))
		return nil, err
	}

	w, err := describe.InitializeLambdaDescribeWorker(
		ctx,
		input.WorkspaceId,
		input.ConnectionId,
		input.ResourceType,
		input.DescribeEndpoint,
		input.KeyARN,
		logger,
	)
	return &events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       req.Body,
	}, nil

	//if err != nil {
	//	logger.Error("Failed to initialize lambda describe worker", zap.Error(err))
	//	return nil, err
	//}
}

func main() {
	lambda.Start(DescribeHandler)
}
