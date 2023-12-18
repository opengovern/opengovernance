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
	"strings"
)

type CreateIngestionPipeline struct {
	securityGroupID string
	subnetID        string
	db              *db.Database
	cfg             config.Config
	osis            *osis.Client
}

func NewCreateIngestionPipeline(
	securityGroupID string,
	subnetID string,
	db *db.Database,
	osis *osis.Client,
	cfg config.Config,
) *CreateIngestionPipeline {
	return &CreateIngestionPipeline{
		securityGroupID: securityGroupID,
		subnetID:        subnetID,
		db:              db,
		osis:            osis,
		cfg:             cfg,
	}
}

func (t *CreateIngestionPipeline) Requirements() []api.TransactionID {
	return []api.TransactionID{api.Transaction_CreateOpenSearch}
}

func (t *CreateIngestionPipeline) Apply(workspace db.Workspace) error {
	processing, endpoint, err := t.isPipelineCreationFinished(workspace)
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			if err := t.createPipeline(workspace); err != nil {
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

func (t *CreateIngestionPipeline) Rollback(workspace db.Workspace) error {
	pipelineName := fmt.Sprintf("kaytu-%s", workspace.ID)

	pipe, err := t.osis.GetPipeline(context.Background(), &osis.GetPipelineInput{
		PipelineName: aws.String(pipelineName),
	})
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return nil
		}
		return err
	}

	deleted := pipe.Pipeline.Status != types.PipelineStatusDeleting
	if !deleted {
		_, err := t.osis.DeletePipeline(context.Background(), &osis.DeletePipelineInput{PipelineName: aws.String(pipelineName)})
		if err != nil {
			return err
		}
		return ErrTransactionNeedsTime
	}

	return nil
}

func (t *CreateIngestionPipeline) isPipelineCreationFinished(workspace db.Workspace) (bool, string, error) {
	pipelineName := fmt.Sprintf("kaytu-%s", workspace.ID)
	pipe, err := t.osis.GetPipeline(context.Background(), &osis.GetPipelineInput{
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

func (t *CreateIngestionPipeline) createPipeline(workspace db.Workspace) error {
	pipelineName := fmt.Sprintf("kaytu-%s", workspace.ID)
	_, err := t.osis.CreatePipeline(context.Background(), &osis.CreatePipelineInput{
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
        # dlq:
          # s3:
            # Provide an S3 bucket
            # bucket: "your-dlq-bucket-name"
            # Provide a key path prefix for the failed requests
            # key_path_prefix: "log-pipeline/logs/dlq"
            # Provide the region of the bucket.
            # region: "us-east-1"
            # Provide a Role ARN with access to the bucket. This role should have a trust relationship with osis-pipelines.amazonaws.com
            # sts_role_arn: "arn:aws:iam::123456789012:role/Example-Role"
`, workspace.OpenSearchEndpoint, fmt.Sprintf("arn:aws:iam::%s:role/kaytu-opensearch-master-%s", t.cfg.AWSAccountID, workspace.ID))),
		PipelineName: aws.String(pipelineName),
		BufferOptions: &types.BufferOptions{
			PersistentBufferEnabled: aws.Bool(true),
		},
		EncryptionAtRestOptions: &types.EncryptionAtRestOptions{KmsKeyArn: nil},
		LogPublishingOptions:    &types.LogPublishingOptions{IsLoggingEnabled: aws.Bool(false)},
		Tags:                    nil,
		VpcOptions: &types.VpcOptions{
			SubnetIds:        []string{t.subnetID},
			SecurityGroupIds: []string{t.securityGroupID},
		},
	})
	if err != nil {
		return err
	}

	return ErrTransactionNeedsTime
}
