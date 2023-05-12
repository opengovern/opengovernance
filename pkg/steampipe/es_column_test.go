package steampipe

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/backup/types"
	"github.com/kaytu-io/kaytu-aws-describer/aws/model"
	"github.com/stretchr/testify/require"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
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

//
//func TestJSONMarshal(t *testing.T) {
//	tim := time.Now()
//	desc := keibi.BackupProtectedResource{
//		Metadata: model.Metadata{
//			Partition: "partition",
//			Region:    "region",
//			AccountID: "accountID",
//		},
//		Description: model.BackupProtectedResourceDescription{
//			ProtectedResource: types.ProtectedResource{
//				LastBackupTime: timeptr(tim),
//				ResourceArn:    strptr("resource_arn"),
//				ResourceType:   strptr("resource_type"),
//			},
//		},
//	}
//	_, err := JSONMarshal(reflect.ValueOf(desc))
//	require.NoError(t, err)
//
//	desc2 := keibi.StorageContainer{
//		Metadata: azureModel.Metadata{},
//		Description: azureModel.StorageContainerDescription{
//			ListContainerItem:  storage.ListContainerItem{
//				ContainerProperties: &storage.ContainerProperties{
//					LastModifiedTime:            &date.Time{Time: tim},
//				},
//			},
//		},
//	}
//	r, err := JSONMarshal(reflect.ValueOf(desc2))
//	require.NoError(t, err)
//
//	j, err := json.Marshal(r)
//	require.NoError(t, err)
//
//	j0, err := json.Marshal(desc2)
//	require.NoError(t, err)
//
//	require.Equal(t, string(j), string(j0))
//}
//
