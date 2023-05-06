package vault

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-aws-describer/aws"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type KMSVaultSourceConfig struct {
	kmsClient *kms.Client
}

func NewKMSVaultSourceConfig(ctx context.Context, accessKey, secretKey, region string) (*KMSVaultSourceConfig, error) {
	cfg, err := aws.GetConfig(ctx, accessKey, secretKey, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to load SDK configuration: %v", err)
	}

	cfg.Region = region
	// Create KMS client with loaded configuration
	svc := kms.NewFromConfig(cfg)

	return &KMSVaultSourceConfig{
		kmsClient: svc,
	}, nil
}

func (v *KMSVaultSourceConfig) Encrypt(cred map[string]any, keyARN string) ([]byte, error) {
	bytes, err := json.Marshal(cred)
	if err != nil {
		return nil, err
	}

	result, err := v.kmsClient.Encrypt(context.TODO(), &kms.EncryptInput{
		KeyId:               &keyARN,
		Plaintext:           bytes,
		EncryptionAlgorithm: types.EncryptionAlgorithmSpecSymmetricDefault,
		EncryptionContext:   nil, //TODO-Saleh use workspaceID
		GrantTokens:         nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt ciphertext: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(result.CiphertextBlob)
	return []byte(encoded), nil
}

func (v *KMSVaultSourceConfig) Decrypt(cypherText string, keyARN string) (map[string]any, error) {
	bytes, err := base64.StdEncoding.DecodeString(cypherText)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %v", err)
	}

	result, err := v.kmsClient.Decrypt(context.TODO(), &kms.DecryptInput{
		CiphertextBlob:      bytes,
		EncryptionAlgorithm: types.EncryptionAlgorithmSpecSymmetricDefault,
		KeyId:               &keyARN,
		EncryptionContext:   nil, //TODO-Saleh use workspaceID
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt ciphertext: %v", err)
	}

	conf := make(map[string]any)
	err = json.Unmarshal(result.Plaintext, &conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
