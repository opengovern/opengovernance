package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	describePkg "github.com/kaytu-io/kaytu-engine/pkg/describe"
	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	inventory "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/meta"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/integration/repository"
	"github.com/kaytu-io/kaytu-util/pkg/fp"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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

func (h Credential) UpdateAzure(ctx context.Context, id uuid.UUID, req entity.UpdateCredentialRequest) error {
	ctx, span := h.tracer.Start(ctx, "update-aws-credential")
	defer span.End()

	cred, err := h.Get(ctx, id.String())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("credential name", *cred.Name),
	))

	if req.Name != nil {
		cred.Name = req.Name
	}

	cnf, err := h.kms.Decrypt(cred.Secret, h.keyARN)
	if err != nil {
		return err
	}
	config, err := fp.FromMap[describePkg.AzureSubscriptionConfig](cnf)
	if err != nil {
		return err
	}

	if req.Config != nil {
		configStr, err := json.Marshal(req.Config)
		if err != nil {
			return err
		}

		var newConfig api.AzureCredentialConfig

		if err := json.Unmarshal(configStr, &newConfig); err != nil {
			return err
		}

		if newConfig.SubscriptionId != "" {
			config.SubscriptionID = newConfig.SubscriptionId
		}

		if newConfig.TenantId != "" {
			config.TenantID = newConfig.TenantId
		}

		if newConfig.ObjectId != "" {
			config.ObjectID = newConfig.ObjectId
		}

		if newConfig.SecretId != "" {
			config.SecretID = newConfig.SecretId
		}

		if newConfig.ClientId != "" {
			config.ClientID = newConfig.ClientId
		}

		if newConfig.ClientSecret != "" {
			config.ClientSecret = newConfig.ClientSecret
		}
	}

	metadata, err := h.AzureMetadata(ctx, *config)
	if err != nil {
		return err
	}

	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	cred.Metadata = jsonMetadata
	secretBytes, err := h.kms.Encrypt(config.ToMap(), h.keyARN)
	if err != nil {
		return err
	}

	cred.Secret = string(secretBytes)
	if metadata.SpnName != "" {
		cred.Name = &metadata.SpnName
	}

	if err := h.repo.Update(ctx, cred); err != nil {
		return err
	}

	if err := h.repo.Update(ctx, cred); err != nil {
		return err
	}

	return nil
}
