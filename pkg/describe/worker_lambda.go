package describe

import (
	"context"
	"fmt"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"go.uber.org/zap"
)

type LambdaDescribeWorker struct {
	workspaceId      string
	connectionId     string
	resourceType     string
	describeEndpoint string
	vault            *vault.KMSVaultSourceConfig
	job              DescribeJob
	logger           *zap.Logger
}

func InitializeLambdaDescribeWorker(
	ctx context.Context,
	workspaceId string,
	connectionId string,
	resourceType string,
	describeEndpoint string,
	keyARN string,
	describeJob DescribeJob,
	logger *zap.Logger,
) (w *LambdaDescribeWorker, err error) {
	if workspaceId == "" {
		return nil, fmt.Errorf("'workspaceId' must be set to a non empty string")
	}
	if connectionId == "" {
		return nil, fmt.Errorf("'connectionId' must be set to a non empty string")
	}
	if resourceType == "" {
		return nil, fmt.Errorf("'resourceType' must be set to a non empty string")
	}
	if describeEndpoint == "" {
		return nil, fmt.Errorf("'describeEndpoint' must be set to a non empty string")
	}

	kmsVault, err := vault.NewKMSVaultSourceConfig(ctx, keyARN)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize KMS vault: %w", err)
	}

	w = &LambdaDescribeWorker{
		workspaceId:      workspaceId,
		connectionId:     connectionId,
		resourceType:     resourceType,
		describeEndpoint: describeEndpoint,
		vault:            kmsVault,
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

	w.job.Do(ctx, w.vault, nil, w.logger, &w.describeEndpoint)

	return nil
}

func (w *LambdaDescribeWorker) Stop() {
	return
}
