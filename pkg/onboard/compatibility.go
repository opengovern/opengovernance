package onboard

import (
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	apiv2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api/v2"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"golang.org/x/net/context"
)

func (h HttpHandler) CredentialV2ToV1(ctx context.Context, newCred model.Credential) (string, error) {
	cnf, err := h.kms.Decrypt(ctx, newCred.Secret, newCred.CredentialStoreKeyID, newCred.CredentialStoreKeyVersion)
	if err != nil {
		return "", err
	}

	awsCnf, err := apiv2.AWSCredentialV2ConfigFromMap(cnf)
	if err != nil {
		return "", err
	}

	newConf := api.AWSCredentialConfig{
		AccountId:            awsCnf.AccountID,
		Regions:              nil,
		AccessKey:            h.masterAccessKey,
		SecretKey:            h.masterSecretKey,
		AssumeRoleName:       awsCnf.AssumeRoleName,
		AssumeAdminRoleName:  awsCnf.AssumeRoleName,
		AssumeRolePolicyName: "",
		ExternalId:           awsCnf.ExternalId,
	}

	latestVersion, err := h.kms.GetLatestVersion(ctx, h.vaultKeyId)
	if err != nil {
		return "", err
	}

	newSecret, err := h.kms.Encrypt(ctx, newConf.AsMap(), h.vaultKeyId, latestVersion)
	if err != nil {
		return "", err
	}

	return string(newSecret), nil
}
