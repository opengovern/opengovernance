package resources

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	types3 "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"strings"
)

type opensearchDomain struct {
	opensearch *opensearch.Client
	db         *db.Database

	domainName      string
	masterRoleARN   string
	instanceType    types3.OpenSearchPartitionInstanceType
	instanceCount   int32
	securityGroupID string
	subnetID        string
	workspaceID     string
}

func OpenSearchDomain(opensearch *opensearch.Client, db *db.Database) *opensearchDomain {
	return &opensearchDomain{
		opensearch: opensearch,
		db:         db,
	}
}

func (domain *opensearchDomain) WithName(name string) *opensearchDomain {
	domain.domainName = name
	return domain
}

func (domain *opensearchDomain) WithMasterRoleARN(roleARN string) *opensearchDomain {
	domain.masterRoleARN = roleARN
	return domain
}

func (domain *opensearchDomain) WithInstanceType(instanceType types3.OpenSearchPartitionInstanceType) *opensearchDomain {
	domain.instanceType = instanceType
	return domain
}

func (domain *opensearchDomain) WithInstanceCount(instanceCount int32) *opensearchDomain {
	domain.instanceCount = instanceCount
	return domain
}

func (domain *opensearchDomain) WithSecurityGroupID(securityGroupID string) *opensearchDomain {
	domain.securityGroupID = securityGroupID
	return domain
}

func (domain *opensearchDomain) WithSubnetID(subnetID string) *opensearchDomain {
	domain.subnetID = subnetID
	return domain
}

func (domain *opensearchDomain) WithWorkspaceID(workspaceID string) *opensearchDomain {
	domain.workspaceID = workspaceID
	return domain
}

func (domain *opensearchDomain) CreateIdempotent(ctx context.Context) error {
	d, err := domain.opensearch.DescribeDomain(ctx, &opensearch.DescribeDomainInput{
		DomainName: aws.String(domain.domainName),
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			_, err = domain.opensearch.CreateDomain(ctx, &opensearch.CreateDomainInput{
				DomainName:     aws.String(domain.domainName),
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
						MasterUserARN:      aws.String(domain.masterRoleARN),
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
					InstanceCount:             aws.Int32(domain.instanceCount),
					InstanceType:              domain.instanceType,
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
					KmsKeyId: nil,
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
					SecurityGroupIds: []string{domain.securityGroupID},
					SubnetIds:        []string{domain.subnetID},
				},
			})
			if err != nil {
				return err
			}
			return ErrResourceNeedsTime
		}
	}

	processing := true
	if d.DomainStatus.Processing != nil {
		processing = *d.DomainStatus.Processing
	}
	if processing {
		return ErrResourceNeedsTime
	}

	var endpoint string
	for k, v := range d.DomainStatus.Endpoints {
		if k == "vpc" {
			endpoint = "https://" + v
		} else {
			endpoint = v
		}
	}
	if endpoint == "" {
		return ErrResourceNeedsTime
	}

	err = domain.db.UpdateWorkspaceOpenSearchEndpoint(domain.workspaceID, endpoint)
	if err != nil {
		return err
	}

	return nil
}

func (domain *opensearchDomain) DeleteIdempotent(ctx context.Context) error {
	d, err := domain.opensearch.DescribeDomain(ctx, &opensearch.DescribeDomainInput{
		DomainName: aws.String(domain.domainName),
	})
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return nil
		}
		return err
	}

	deleted := false
	if d.DomainStatus.Deleted != nil {
		deleted = *d.DomainStatus.Deleted
	}

	if !deleted {
		_, err := domain.opensearch.DeleteDomain(ctx, &opensearch.DeleteDomainInput{DomainName: aws.String(domain.domainName)})
		if err != nil {
			return err
		}
		return ErrResourceNeedsTime
	}

	processing := false
	if d.DomainStatus.Processing != nil {
		processing = *d.DomainStatus.Processing
	}

	if processing {
		return ErrResourceNeedsTime
	}

	err = domain.db.UpdateWorkspaceOpenSearchEndpoint(domain.workspaceID, "")
	if err != nil {
		return err
	}
	return nil
}
