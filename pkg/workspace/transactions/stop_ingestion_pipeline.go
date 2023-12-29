package transactions

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/osis"
	"github.com/aws/aws-sdk-go-v2/service/osis/types"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
)

type StopIngestionPipeline struct {
	cfg  config.Config
	osis *osis.Client
}

func NewStopIngestionPipeline(
	cfg config.Config,
	osis *osis.Client,
) *StopIngestionPipeline {
	return &StopIngestionPipeline{
		cfg:  cfg,
		osis: osis,
	}
}

func (t *StopIngestionPipeline) Requirements() []api.TransactionID {
	return []api.TransactionID{api.Transaction_CreateIngestionPipeline}
}

func (t *StopIngestionPipeline) ApplyIdempotent(workspace db.Workspace) error {
	pipelineName := fmt.Sprintf("kaytu-%s", workspace.ID)
	pipeline, err := t.osis.GetPipeline(context.Background(), &osis.GetPipelineInput{PipelineName: aws.String(pipelineName)})
	if err != nil {
		return err
	}

	if pipeline.Pipeline.Status == types.PipelineStatusStopped {
		return nil
	}

	if pipeline.Pipeline.Status == types.PipelineStatusStopping {
		return ErrTransactionNeedsTime
	}

	_, err = t.osis.StopPipeline(context.Background(), &osis.StopPipelineInput{PipelineName: aws.String(pipelineName)})
	if err != nil {
		return err
	}

	return ErrTransactionNeedsTime
}

func (t *StopIngestionPipeline) RollbackIdempotent(workspace db.Workspace) error {
	pipelineName := fmt.Sprintf("kaytu-%s", workspace.ID)
	pipeline, err := t.osis.GetPipeline(context.Background(), &osis.GetPipelineInput{PipelineName: aws.String(pipelineName)})
	if err != nil {
		return err
	}

	if pipeline.Pipeline.Status == types.PipelineStatusActive {
		return nil
	}

	if pipeline.Pipeline.Status == types.PipelineStatusStarting {
		fmt.Println("StopIngestionPipeline -> RollbackIdempotent -> Starting")
		return ErrTransactionNeedsTime
	}

	_, err = t.osis.StartPipeline(context.Background(), &osis.StartPipelineInput{PipelineName: aws.String(pipelineName)})
	if err != nil {
		return err
	}

	fmt.Println("StopIngestionPipeline -> RollbackIdempotent -> StartPipeline")
	return ErrTransactionNeedsTime
}
