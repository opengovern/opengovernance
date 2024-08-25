package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type AzureCredentialMetadata struct {
	SpnName              string    `json:"spn_name"`
	ObjectId             string    `json:"object_id"`
	SecretId             string    `json:"secret_id"`
	SecretExpirationDate time.Time `json:"secret_expiration_date"`
}

func (m AzureCredentialMetadata) GetExpirationDate() time.Time {
	return m.SecretExpirationDate
}

func NewAzureCredential(credentialType CredentialType, metadata *AzureCredentialMetadata) (*Credential, error) {
	id := uuid.New()
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	crd := &Credential{
		ID:             id,
		Name:           nil,
		ConnectorType:  source.CloudAzure,
		Secret:         fmt.Sprintf("sources/%s/%s", strings.ToLower(string(source.CloudAzure)), id),
		CredentialType: credentialType,
		Metadata:       jsonMetadata,
	}
	if credentialType == CredentialTypeManualAzureSpn || credentialType == CredentialTypeManualAzureEntraId {
		crd.AutoOnboardEnabled = true
	}

	return crd, nil
}

type AzureSubscription struct {
	SubscriptionID string
	SubModel       armsubscription.Subscription
	SubTags        []armresources.TagDetails
}

// AzureConnectionMetadata converts into json and stored along side its connection.
type AzureConnectionMetadata struct {
	TenantID       string                       `json:"tenant_id"`
	SubscriptionID string                       `json:"subscription_id"`
	SubModel       armsubscription.Subscription `json:"subscription_model"`
	SubTags        map[string][]string          `json:"subscription_tags"`
}

func NewAzureConnectionMetadata(
	sub *AzureSubscription,
	tenantID string,
) AzureConnectionMetadata {
	metadata := AzureConnectionMetadata{
		SubscriptionID: sub.SubscriptionID,
		SubModel:       sub.SubModel,
		SubTags:        make(map[string][]string),
		TenantID:       tenantID,
	}
	for _, tag := range sub.SubTags {
		if tag.TagName == nil || tag.Count == nil {
			continue
		}
		metadata.SubTags[*tag.TagName] = make([]string, 0, len(tag.Values))
		for _, value := range tag.Values {
			if value == nil || value.TagValue == nil {
				continue
			}
			metadata.SubTags[*tag.TagName] = append(metadata.SubTags[*tag.TagName], *value.TagValue)
		}
	}

	return metadata
}
