package onboard

import (
	"github.com/opengovern/opengovernance/pkg/onboard/api"
	apiv2 "github.com/opengovern/opengovernance/pkg/onboard/api/v2"
	"github.com/opengovern/opengovernance/services/integration/model"
	"golang.org/x/net/context"
)

func (h HttpHandler) CredentialV2ToV1(ctx context.Context, newCred model.Credential) (string, error) {
	cnf, err := h.vaultSc.Decrypt(ctx, newCred.Secret)
	if err != nil {
		return "", err
	}

	awsCnf, err := apiv2.AWSCredentialV2ConfigFromMap(cnf)
	if err != nil {
		return "", err
	}

	aKey := h.masterAccessKey
	sKey := h.masterSecretKey
	if awsCnf.AccessKey != nil {
		aKey = *awsCnf.AccessKey
	}
	if awsCnf.SecretKey != nil {
		sKey = *awsCnf.SecretKey
	}

	newConf := api.AWSCredentialConfig{
		AccountId:           awsCnf.AccountID,
		Regions:             nil,
		AccessKey:           aKey,
		SecretKey:           sKey,
		AssumeRoleName:      awsCnf.AssumeRoleName,
		AssumeAdminRoleName: awsCnf.AssumeRoleName,
		ExternalId:          awsCnf.ExternalId,
	}

	newSecret, err := h.vaultSc.Encrypt(ctx, newConf.AsMap())
	if err != nil {
		return "", err
	}

	return string(newSecret), nil
}
