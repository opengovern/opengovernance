package transactions

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/osis"
	"github.com/aws/aws-sdk-go-v2/service/osis/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"strings"
)

type CreateIngestionPipeline struct {
	securityGroupID string
	subnetID        string
	db              *db.Database
	cfg             config.Config
	osis            *osis.Client
	iam             *iam.Client
	s3Client        *s3.Client
}

func NewCreateIngestionPipeline(
	securityGroupID string,
	subnetID string,
	db *db.Database,
	osis *osis.Client,
	iam *iam.Client,
	cfg config.Config,
	s3Client *s3.Client,
) *CreateIngestionPipeline {
	return &CreateIngestionPipeline{
		securityGroupID: securityGroupID,
		subnetID:        subnetID,
		db:              db,
		osis:            osis,
		iam:             iam,
		cfg:             cfg,
		s3Client:        s3Client,
	}
}

func (t *CreateIngestionPipeline) Requirements() []api.TransactionID {
	return []api.TransactionID{api.Transaction_CreateOpenSearch}
}

func (t *CreateIngestionPipeline) ApplyIdempotent(ctx context.Context, workspace db.Workspace) error {
	processing, endpoint, err := t.isPipelineCreationFinished(ctx, workspace)
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			if err := t.createPipeline(ctx, workspace); err != nil {
				return err
			}
			return ErrTransactionNeedsTime
		}
		return err
	}

	if processing {
		return ErrTransactionNeedsTime
	}

	if endpoint == "" {
		return ErrTransactionNeedsTime
	}

	err = t.db.UpdateWorkspacePipelineEndpoint(workspace.ID, endpoint)
	if err != nil {
		return err
	}

	return nil
}

func (t *CreateIngestionPipeline) RollbackIdempotent(ctx context.Context, workspace db.Workspace) error {
	pipelineName := fmt.Sprintf("kaytu-%s", workspace.ID)

	pipe, err := t.osis.GetPipeline(ctx, &osis.GetPipelineInput{
		PipelineName: aws.String(pipelineName),
	})
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return nil
		}
		return err
	}

	if pipe.Pipeline.Status != types.PipelineStatusDeleting {
		_, err := t.osis.DeletePipeline(ctx, &osis.DeletePipelineInput{PipelineName: aws.String(pipelineName)})
		if err != nil {
			return err
		}
		return ErrTransactionNeedsTime
	}

	return nil
}

func (t *CreateIngestionPipeline) isPipelineCreationFinished(ctx context.Context, workspace db.Workspace) (bool, string, error) {
	pipelineName := fmt.Sprintf("kaytu-%s", workspace.ID)
	pipe, err := t.osis.GetPipeline(ctx, &osis.GetPipelineInput{
		PipelineName: aws.String(pipelineName),
	})
	if err != nil {
		return false, "", err
	}

	processing := pipe.Pipeline.Status != types.PipelineStatusActive

	var endpoint string
	for _, v := range pipe.Pipeline.IngestEndpointUrls {
		endpoint = v
	}
	return processing, endpoint, nil
}

func (t *CreateIngestionPipeline) createPipeline(ctx context.Context, workspace db.Workspace) error {
	pipelineName := fmt.Sprintf("kaytu-%s", workspace.ID)
	roleARN := fmt.Sprintf("arn:aws:iam::%s:role/kaytu-opensearch-master-%s", t.cfg.AWSAccountID, workspace.ID)
	bucketName := fmt.Sprintf("dlq-%s", workspace.ID)
	_, err := t.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		var bucketAlreadyExists *s3Types.BucketAlreadyExists
		if errors.As(err, &bucketAlreadyExists) {
			return nil
		}
		return err
	}

	_, err = t.osis.CreatePipeline(ctx, &osis.CreatePipelineInput{
		MaxUnits: aws.Int32(1),
		MinUnits: aws.Int32(1),
		PipelineConfigurationBody: aws.String(fmt.Sprintf(`version: "2"
resource-sink:
  source:
    http:
      # Provide the path for ingestion. ${pipelineName} will be replaced with sub-pipeline name, i.e. log-pipeline, configured for this pipeline.
      # In this case it would be "/log-pipeline/logs". This will be the FluentBit output URI value.
      path: "/resource-sink"
  processor:
  sink:
    - opensearch:
        # Provide an AWS OpenSearch Service domain endpoint
        hosts: [ "%[1]s" ]
        aws:
          # Provide a Role ARN with access to the domain. This role should have a trust relationship with osis-pipelines.amazonaws.com
          sts_role_arn: "%[2]s"
          # Provide the region of the domain.
          region: "us-east-2"
          # Enable the 'serverless' flag if the sink is an Amazon OpenSearch Serverless collection
          # serverless: true
        index: "${es_index}"
        document_id: "${es_id}"
        # Enable the 'distribution_version' setting if the AWS OpenSearch Service domain is of version Elasticsearch 6.x
        # distribution_version: "es6"
        # Enable and switch the 'enable_request_compression' flag if the default compression setting is changed in the domain. See https://docs.aws.amazon.com/opensearch-service/latest/developerguide/gzip.html
        # enable_request_compression: true/false
        # Enable the S3 DLQ to capture any failed requests in an S3 bucket
        dlq:
          s3:
            # Provide an S3 bucket
            bucket: "%[3]s"
            # Provide a key path prefix for the failed requests
            key_path_prefix: "log-pipeline/logs/dlq"
            # Provide the region of the bucket.
            region: "us-east-2"
            # Provide a Role ARN with access to the bucket. This role should have a trust relationship with osis-pipelines.amazonaws.com
            sts_role_arn: "%[2]s"
`, workspace.OpenSearchEndpoint, roleARN, bucketName)),
		PipelineName: aws.String(pipelineName),
		BufferOptions: &types.BufferOptions{
			PersistentBufferEnabled: aws.Bool(false),
		},
		//EncryptionAtRestOptions: &types.EncryptionAtRestOptions{KmsKeyArn: nil},
		LogPublishingOptions: &types.LogPublishingOptions{IsLoggingEnabled: aws.Bool(false)},
		Tags:                 nil,
		VpcOptions:           nil,
	})
	if err != nil {
		return err
	}

	return ErrTransactionNeedsTime
}
