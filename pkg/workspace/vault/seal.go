package vault

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"time"
)

type SealHandler struct {
	logger           *zap.Logger
	cfg              config.Config
	vaultSealHandler *vault.HashiCorpVaultSealHandler
	kubeClientset    *kubernetes.Clientset
}

func NewSealHandler(ctx context.Context, logger *zap.Logger, cfg config.Config) (*SealHandler, error) {
	hashiCorpVaultSealHandler, err := vault.NewHashiCorpVaultSealHandler(ctx, logger, cfg.Vault.HashiCorp)
	if err != nil {
		logger.Error("new hashicorp vaultClient seal handler", zap.Error(err))
		return nil, fmt.Errorf("new hashicorp vaultClient seal handler: %w", err)
	}

	kuberConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Error("failed to get kubernetes config", zap.Error(err))
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(kuberConfig)
	if err != nil {
		logger.Error("failed to create kubernetes clientset", zap.Error(err))
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &SealHandler{
		logger:           logger,
		cfg:              cfg,
		vaultSealHandler: hashiCorpVaultSealHandler,
		kubeClientset:    clientset,
	}, nil
}

const (
	secretName = "vault-unseal-keys"
)

func (s *SealHandler) initVault(ctx context.Context) {
	initRes, err := s.vaultSealHandler.TryInit(ctx)
	if err != nil {
		s.logger.Fatal("failed to init vault", zap.Error(err))
	}
	if initRes == nil {
		s.logger.Info("vault already initialized")
	} else {
		keysSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: s.cfg.KaytuOctopusNamespace,
			},
			StringData: make(map[string]string),
		}

		for i, key := range initRes.Keys {
			keysSecret.StringData[fmt.Sprintf("key-%d", i)] = key
		}
		keysSecret.StringData["root-token"] = initRes.RootToken

		_, err = s.kubeClientset.CoreV1().Secrets(s.cfg.KaytuOctopusNamespace).Create(ctx, &keysSecret, metav1.CreateOptions{})
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			s.logger.Fatal("failed to create vault unseal keys secret", zap.Error(err), zap.Strings("keys", initRes.Keys))
		} else if k8serrors.IsAlreadyExists(err) {
			_, err := s.kubeClientset.CoreV1().Secrets(s.cfg.KaytuOctopusNamespace).Update(ctx, &keysSecret, metav1.UpdateOptions{})
			if err != nil {
				s.logger.Fatal("failed to update vault unseal keys secret", zap.Error(err), zap.Strings("keys", initRes.Keys))
			}
		}
	}
}

func (s *SealHandler) unsealChecker(ctx context.Context, unsealed chan<- struct{}) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("unsealChecker panic", zap.Any("recover", r))
			go s.unsealChecker(ctx, unsealed)
		}
	}()

	keysSecret, err := s.kubeClientset.CoreV1().Secrets(s.cfg.KaytuOctopusNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("failed to get vault unseal keys secret", zap.Error(err))
		return
	}
	keys := make([]string, 0, len(keysSecret.Data))
	for _, v := range keysSecret.Data {
		decodedV, err := base64.StdEncoding.DecodeString(string(v))
		if err != nil {
			s.logger.Error("failed to decode unseal key", zap.Error(err))
			return
		}
		keys = append(keys, string(decodedV))
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	err = s.vaultSealHandler.TryUnseal(ctx, keys)
	if err != nil {
		s.logger.Error("failed to unseal vault", zap.Error(err))
	}
	if unsealed != nil && err == nil {
		unsealed <- struct{}{}
		close(unsealed)
		unsealed = nil
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err = s.vaultSealHandler.TryUnseal(ctx, keys)
			if err != nil {
				s.logger.Error("failed to unseal vault", zap.Error(err))
				continue
			}
			if unsealed != nil {
				unsealed <- struct{}{}
				close(unsealed)
				unsealed = nil
			}
		}
	}
}

func (s *SealHandler) Start(ctx context.Context) {
	s.initVault(ctx)
	unsealChan := make(chan struct{})
	go s.unsealChecker(ctx, unsealChan)
	// block until vault is unsealed
	<-unsealChan
	s.logger.Info("vault unsealed")
}
