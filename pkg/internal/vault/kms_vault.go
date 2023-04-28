package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type KMSVaultSourceConfig struct {
	kmsClient *kms.Client
	keyARN    string
}

func NewKMSVaultSourceConfig(ctx context.Context, keyARN string) (*KMSVaultSourceConfig, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load SDK configuration: %v", err)
	}

	// Create KMS client with loaded configuration
	svc := kms.NewFromConfig(cfg)

	return &KMSVaultSourceConfig{
		kmsClient: svc,
		keyARN:    keyARN,
	}, nil
}

func (v *KMSVaultSourceConfig) Write(pathRef string, config map[string]any) error {
	return errors.New("writing is not supported")
}

func (v *KMSVaultSourceConfig) Read(cypherText string) (map[string]any, error) {
	result, err := v.kmsClient.Decrypt(context.TODO(), &kms.DecryptInput{
		CiphertextBlob:      []byte(cypherText),
		EncryptionAlgorithm: types.EncryptionAlgorithmSpecSymmetricDefault,
		KeyId:               &v.keyARN,
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

func (v *KMSVaultSourceConfig) Delete(pathRef string) error {
	return errors.New("deleting is not supported")
}
