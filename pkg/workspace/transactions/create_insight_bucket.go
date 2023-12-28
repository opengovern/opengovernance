package transactions

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
)

type CreateInsightBucket struct {
	s3Client *s3.Client
}

func NewCreateInsightBucket(
	s3Client *s3.Client,
) *CreateInsightBucket {
	return &CreateInsightBucket{
		s3Client: s3Client,
	}
}

func (t *CreateInsightBucket) Requirements() []api.TransactionID {
	return nil
}

func (t *CreateInsightBucket) ApplyIdempotent(workspace db.Workspace) error {
	bucketName := fmt.Sprintf("insights-%s", workspace.ID)
	_, err := t.s3Client.CreateBucket(context.Background(), &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	var bucketAlreadyExists *s3Types.BucketAlreadyExists
	if errors.As(err, &bucketAlreadyExists) {
		return nil
	}
	return err
}

func (t *CreateInsightBucket) RollbackIdempotent(workspace db.Workspace) error {
	bucketName := fmt.Sprintf("insights-%s", workspace.ID)
	objects, err := t.s3Client.ListObjects(context.Background(), &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		var noSuchBucket *s3Types.NoSuchBucket
		if errors.As(err, &noSuchBucket) {
			return nil
		}
		return err
	}

	var objs []s3Types.ObjectIdentifier
	for _, obj := range objects.Contents {
		objs = append(objs, s3Types.ObjectIdentifier{
			Key: obj.Key,
		})
	}
	if len(objs) > 0 {
		_, err = t.s3Client.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
			Bucket: aws.String(bucketName),
			Delete: &s3Types.Delete{
				Objects: objs,
			},
		})
		if err != nil {
			return err
		}
	}

	_, err = t.s3Client.DeleteBucket(context.Background(), &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	return err
}
