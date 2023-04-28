package describe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"go.uber.org/zap"
)

type LambdaDescribeWorkerInput struct {
	WorkspaceId      string      `json:"workspaceId"`
	DescribeEndpoint string      `json:"describeEndpoint"`
	KeyARN           string      `json:"keyARN"`
	DescribeJob      DescribeJob `json:"describeJob"`
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

	w, err := InitializeLambdaDescribeWorker(
		ctx,
		input.WorkspaceId,
		input.DescribeEndpoint,
		input.KeyARN,
		input.DescribeJob,
		logger,
	)

	return w.Run(ctx)
}

type LambdaDescribeWorker struct {
	workspaceId      string
	describeEndpoint string
	vault            *vault.KMSVaultSourceConfig
	keyARN           string
	job              DescribeJob
	logger           *zap.Logger
}

func InitializeLambdaDescribeWorker(
	ctx context.Context,
	workspaceId string,
	describeEndpoint string,
	keyARN string,
	describeJob DescribeJob,
	logger *zap.Logger,
) (w *LambdaDescribeWorker, err error) {
	if workspaceId == "" {
		return nil, fmt.Errorf("'workspaceId' must be set to a non empty string")
	}
	if describeEndpoint == "" {
		return nil, fmt.Errorf("'describeEndpoint' must be set to a non empty string")
	}

	kmsVault, err := vault.NewKMSVaultSourceConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize KMS vault: %w", err)
	}

	w = &LambdaDescribeWorker{
		workspaceId:      workspaceId,
		describeEndpoint: describeEndpoint,
		vault:            kmsVault,
		keyARN:           keyARN,
		job:              describeJob,
		logger:           logger,
	}
	defer func() {
		if err != nil && w != nil {
			w.Stop()
		}
	}()

	w.logger = logger

	return w, nil
}

func (w *LambdaDescribeWorker) Run(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	w.job.Do(ctx, w.vault, w.keyARN, nil, w.logger, &w.describeEndpoint)

	return nil
}

func (w *LambdaDescribeWorker) Stop() {
	return
}
