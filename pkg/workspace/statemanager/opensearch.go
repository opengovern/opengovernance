package statemanager

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	types3 "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
)

func (s *Service) isOpenSearchCreationFinished(workspace *db.Workspace) (bool, error) {
	domainName := workspace.ID

	domain, err := s.opensearch.DescribeDomain(context.Background(), &opensearch.DescribeDomainInput{
		DomainName: aws.String(domainName),
	})
	if err != nil {
		return false, err
	}

	processing := true
	if domain.DomainStatus.Processing != nil {
		processing = *domain.DomainStatus.Processing
	}
	return processing, nil
}

func (s *Service) createOpenSearch(workspace *db.Workspace) error {
	domainName := workspace.ID
	masterRoleARN := "arn:aws:iam::435670955331:role/KaytuOpenSearchAdmin"
	vmType := types3.OpenSearchPartitionInstanceTypeT3SmallSearch
	instanceCount := int32(1)
	securityGroupID := "sg-07c5a6f32dcd14e26"
	subnetIDs := []string{"subnet-099c1b7e69b8d4a3f"}

	_, err := s.opensearch.CreateDomain(context.Background(), &opensearch.CreateDomainInput{
		DomainName:     aws.String(domainName),
		AccessPolicies: nil,
		AdvancedOptions: map[string]string{
			"indices.query.bool.max_clause_count":    "1024",
			"override_main_response_version":         "false",
			"rest.action.multi.allow_explicit_index": "true",
			"indices.fielddata.cache.size":           "20",
		},
		AdvancedSecurityOptions: &types3.AdvancedSecurityOptionsInput{
			AnonymousAuthEnabled:        nil,
			Enabled:                     aws.Bool(true),
			InternalUserDatabaseEnabled: nil,
			MasterUserOptions: &types3.MasterUserOptions{
				MasterUserARN:      aws.String(masterRoleARN),
				MasterUserName:     nil,
				MasterUserPassword: nil,
			},
			SAMLOptions: nil,
		},
		AutoTuneOptions: &types3.AutoTuneOptionsInput{DesiredState: types3.AutoTuneDesiredStateDisabled},
		ClusterConfig: &types3.ClusterConfig{
			ColdStorageOptions:        &types3.ColdStorageOptions{Enabled: aws.Bool(false)},
			DedicatedMasterCount:      nil,
			DedicatedMasterEnabled:    aws.Bool(false),
			DedicatedMasterType:       "",
			InstanceCount:             aws.Int32(instanceCount),
			InstanceType:              vmType,
			MultiAZWithStandbyEnabled: aws.Bool(false),
			WarmCount:                 nil,
			WarmEnabled:               aws.Bool(false),
			WarmType:                  "",
			ZoneAwarenessConfig:       nil,
			ZoneAwarenessEnabled:      aws.Bool(false),
		},
		CognitoOptions: nil,
		DomainEndpointOptions: &types3.DomainEndpointOptions{
			CustomEndpoint:               nil,
			CustomEndpointCertificateArn: nil,
			CustomEndpointEnabled:        aws.Bool(false),
			EnforceHTTPS:                 aws.Bool(true),
			TLSSecurityPolicy:            "Policy-Min-TLS-1-0-2019-07",
		},
		EBSOptions: &types3.EBSOptions{
			EBSEnabled: aws.Bool(true),
			Iops:       aws.Int32(3000),
			Throughput: aws.Int32(125),
			VolumeSize: aws.Int32(10),
			VolumeType: "gp3",
		},
		EncryptionAtRestOptions: &types3.EncryptionAtRestOptions{
			Enabled:  aws.Bool(true),
			KmsKeyId: nil, //TODO-Saleh KMS
		},
		EngineVersion:               aws.String("OpenSearch_2.11"),
		LogPublishingOptions:        nil,
		NodeToNodeEncryptionOptions: &types3.NodeToNodeEncryptionOptions{Enabled: aws.Bool(true)},
		OffPeakWindowOptions: &types3.OffPeakWindowOptions{
			Enabled: aws.Bool(true),
			OffPeakWindow: &types3.OffPeakWindow{
				WindowStartTime: &types3.WindowStartTime{
					Hours:   0,
					Minutes: 0,
				},
			},
		},
		SoftwareUpdateOptions: nil,
		TagList:               nil,
		VPCOptions: &types3.VPCOptions{
			SecurityGroupIds: []string{securityGroupID},
			SubnetIds:        subnetIDs,
		},
	})
	if err != nil {
		return err
	}

	return nil
}
