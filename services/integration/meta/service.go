package meta

import (
	"errors"
	"github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"strings"

	metadata "github.com/kaytu-io/kaytu-engine/pkg/metadata/client"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
)

type Meta struct {
	AssetDiscoveryAWSPolicyARNs []string
	SpendDiscoveryAWSPolicyARNs []string

	AssetDiscoveryAzureRoleIDs []string
	SpendDiscoveryAzureRoleIDs []string

	Client metadata.MetadataServiceClient
}

func New(config koanf.KaytuService) (*Meta, error) {
	client := metadata.NewMetadataServiceClient(config.BaseURL)

	ctx := &httpclient.Context{
		UserRole: api.InternalRole,
	}

	awsAssetDiscovery, err := client.GetConfigMetadata(ctx, models.MetadataKeyAssetDiscoveryAWSPolicyARNs)
	if err != nil {
		if !errors.Is(err, metadata.ErrConfigNotFound) {
			return nil, err
		}
		awsAssetDiscovery = &models.StringConfigMetadata{}
	}

	awsSpendDiscovery, err := client.GetConfigMetadata(ctx, models.MetadataKeySpendDiscoveryAWSPolicyARNs)
	if err != nil {
		if !errors.Is(err, metadata.ErrConfigNotFound) {
			return nil, err
		}
		awsSpendDiscovery = &models.StringConfigMetadata{}
	}

	azureAssetDiscovery, err := client.GetConfigMetadata(ctx, models.MetadataKeyAssetDiscoveryAzureRoleIDs)
	if err != nil {
		if !errors.Is(err, metadata.ErrConfigNotFound) {
			return nil, err
		}
		azureAssetDiscovery = &models.StringConfigMetadata{}
	}

	azureSpendDiscovery, err := client.GetConfigMetadata(ctx, models.MetadataKeySpendDiscoveryAzureRoleIDs)
	if err != nil {
		if !errors.Is(err, metadata.ErrConfigNotFound) {
			return nil, err
		}
		azureSpendDiscovery = &models.StringConfigMetadata{}
	}

	// make sure we can cast metadata value into string by checking its type.
	if err := models.HasType(awsAssetDiscovery, models.ConfigMetadataTypeString); err != nil {
		return nil, err
	}

	if err := models.HasType(awsSpendDiscovery, models.ConfigMetadataTypeString); err != nil {
		return nil, err
	}

	if err := models.HasType(azureAssetDiscovery, models.ConfigMetadataTypeString); err != nil {
		return nil, err
	}

	if err := models.HasType(azureSpendDiscovery, models.ConfigMetadataTypeString); err != nil {
		return nil, err
	}

	return &Meta{
		AssetDiscoveryAWSPolicyARNs: strings.Split(awsAssetDiscovery.GetValue().(string), ","),
		SpendDiscoveryAWSPolicyARNs: strings.Split(awsSpendDiscovery.GetValue().(string), ","),
		AssetDiscoveryAzureRoleIDs:  strings.Split(azureAssetDiscovery.GetValue().(string), ","),
		SpendDiscoveryAzureRoleIDs:  strings.Split(azureSpendDiscovery.GetValue().(string), ","),
		Client:                      client,
	}, nil
}
