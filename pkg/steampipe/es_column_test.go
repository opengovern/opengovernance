package steampipe

import (
	"github.com/aws/aws-sdk-go-v2/service/backup/types"
	"github.com/stretchr/testify/require"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

	"testing"
	"time"
)

func strptr(str string) *string {
	return &str
}

func timeptr(time time.Time) *time.Time {
	return &time
}

func TestAWSDescriptionToRecord(t *testing.T) {
	tim := time.Now()
	desc := keibi.BackupProtectedResource{
		Metadata: model.Metadata{
			Partition: "partition",
			Region:    "region",
			AccountID: "accountID",
		},
		Description: model.BackupProtectedResourceDescription{
			ProtectedResource: types.ProtectedResource{
				LastBackupTime: timeptr(tim),
				ResourceArn:    strptr("resource_arn"),
				ResourceType:   strptr("resource_type"),
			},
		},
	}
	record, err := AWSDescriptionToRecord(desc, "aws_backup_protected_resource")
	require.NoError(t, err)

	require.Equal(t, "resource_arn", record["resource_arn"].GetStringValue())
	require.Equal(t, "resource_type", record["resource_type"].GetStringValue())
	require.Equal(t, tim.UnixMilli(), record["last_backup_time"].GetTimestampValue().AsTime().UnixMilli())
	require.Equal(t, "[\"resource_arn\"]", string(record["akas"].GetJsonValue()))
	require.Equal(t, "partition", record["partition"].GetStringValue())
	require.Equal(t, "region", record["region"].GetStringValue())
	require.Equal(t, "accountID", record["account_id"].GetStringValue())
}
