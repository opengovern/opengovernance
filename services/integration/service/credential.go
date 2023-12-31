package service

import (
	"context"

	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	inventory "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/services/integration/meta"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/integration/repository"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Credential struct {
	keyARN            string
	kms               *vault.KMSVaultSourceConfig
	tracer            trace.Tracer
	repo              repository.Credential
	transactionalRepo repository.CredConn
	describe          describe.SchedulerServiceClient
	inventory         inventory.InventoryServiceClient
	meta              *meta.Meta
	masterAccessKey   string
	masterSecretKey   string
	logger            *zap.Logger
	connSvc           Connection
}

func NewCredential(
	repo repository.Credential,
	transactionalRepo repository.CredConn,
	kms *vault.KMSVaultSourceConfig,
	keyARN string,
	describe describe.SchedulerServiceClient,
	inventory inventory.InventoryServiceClient,
	meta *meta.Meta,
	connSvc Connection,
	masterAccessKey string,
	masterSecretKey string,
	logger *zap.Logger,
) Credential {
	return Credential{
		tracer:            otel.GetTracerProvider().Tracer("integration.service.credential"),
		repo:              repo,
		transactionalRepo: transactionalRepo,
		keyARN:            keyARN,
		kms:               kms,
		inventory:         inventory,
		describe:          describe,
		meta:              meta,
		masterAccessKey:   masterAccessKey,
		masterSecretKey:   masterSecretKey,
		connSvc:           connSvc,
		logger:            logger.Named("service").Named("credential"),
	}
}

func (h Credential) Create(ctx context.Context, cred *model.Credential) error {
	return h.repo.Create(ctx, cred)
}

func (h Credential) ListWithFilters(
	ctx context.Context,
	connector source.Type,
	health source.HealthStatus,
	credentialType []model.CredentialType,
) ([]model.Credential, error) {
	ctx, span := h.tracer.Start(ctx, "list-with-filters")
	defer span.End()

	creds, err := h.repo.ListByFilters(ctx, connector, health, credentialType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, err
	}

	return creds, err
}

func (h Credential) Get(ctx context.Context, id string) (*model.Credential, error) {
	ctx, span := h.tracer.Start(ctx, "get")
	defer span.End()

	cred, err := h.repo.Get(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, err
	}

	return cred, nil
}

func (h Credential) Delete(ctx context.Context, cred model.Credential) error {
	ctx, span := h.tracer.Start(ctx, "delete")
	defer span.End()

	if err := h.transactionalRepo.DeleteCredential(ctx, cred); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	return nil
}
