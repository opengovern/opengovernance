package service

import (
	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	inventory "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/services/integration/meta"
	"github.com/kaytu-io/kaytu-engine/services/integration/repository"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Credential struct {
	keyARN          string
	kms             *vault.KMSVaultSourceConfig
	tracer          trace.Tracer
	repo            repository.Credential
	describe        describe.SchedulerServiceClient
	inventory       inventory.InventoryServiceClient
	meta            *meta.Meta
	masterAccessKey string
	masterSecretKey string
	logger          *zap.Logger
}

func NewCredential(
	repo repository.Credential,
	kms *vault.KMSVaultSourceConfig,
	keyARN string,
	describe describe.SchedulerServiceClient,
	inventory inventory.InventoryServiceClient,
	meta *meta.Meta,
	masterAccessKey string,
	masterSecretKey string,
	logger *zap.Logger,
) Credential {
	return Credential{
		tracer:          otel.GetTracerProvider().Tracer("integration.service.credential"),
		repo:            repo,
		keyARN:          keyARN,
		kms:             kms,
		inventory:       inventory,
		describe:        describe,
		meta:            meta,
		masterAccessKey: masterAccessKey,
		masterSecretKey: masterSecretKey,
		logger:          logger.Named("service").Named("credential"),
	}
}
