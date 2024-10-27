package meta

import (
	"errors"
	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"strings"

	"github.com/opengovern/og-util/pkg/koanf"
	metadata "github.com/opengovern/opengovernance/pkg/metadata/client"
	"github.com/opengovern/opengovernance/pkg/metadata/models"
)

type Meta struct {
	AssetDiscoveryAzureRoleIDs []string
	SpendDiscoveryAzureRoleIDs []string

	Client metadata.MetadataServiceClient
}

func New(config koanf.OpenGovernanceService) (*Meta, error) {
	client := metadata.NewMetadataServiceClient(config.BaseURL)

	ctx := &httpclient.Context{
		UserRole: api.AdminRole,
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

	if err := models.HasType(azureAssetDiscovery, models.ConfigMetadataTypeString); err != nil {
		return nil, err
	}

	if err := models.HasType(azureSpendDiscovery, models.ConfigMetadataTypeString); err != nil {
		return nil, err
	}

	return &Meta{
		AssetDiscoveryAzureRoleIDs: strings.Split(azureAssetDiscovery.GetValue().(string), ","),
		SpendDiscoveryAzureRoleIDs: strings.Split(azureSpendDiscovery.GetValue().(string), ","),
		Client:                     client,
	}, nil
}
