package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"go.uber.org/zap"
)

type LambdaDescribeWorkerInput struct {
	WorkspaceId      string               `json:"workspaceId"`
	ConnectionId     string               `json:"connectionId"`
	ResourceType     string               `json:"resourceType"`
	DescribeEndpoint string               `json:"describeEndpoint"`
	KeyARN           string               `json:"keyARN"`
	SecretCypher     string               `json:"secretCypher"`
	DescribeJob      describe.DescribeJob `json:"describeJob"`
}

func DescribeHandler(ctx context.Context, req events.APIGatewayProxyRequest) error {
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}

	logger.Info(req.Body)

	var input LambdaDescribeWorkerInput
	err = json.Unmarshal([]byte(req.Body), &input)
	if err != nil {
		logger.Error("Failed to unmarshal input", zap.Error(err))
		return err
	}

	w, err := describe.InitializeLambdaDescribeWorker(
		ctx,
		input.WorkspaceId,
		input.ConnectionId,
		input.ResourceType,
		input.DescribeEndpoint,
		input.KeyARN,
		input.DescribeJob,
		logger,
	)

	return w.Run(ctx)
}

func main() {
	lambda.Start(DescribeHandler)
}
