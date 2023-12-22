package service

import (
	"context"
	"time"

	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	inventoryAPI "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	inventory "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/integration/repository"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Connection struct {
	keyARN          string
	kms             *vault.KMSVaultSourceConfig
	tracer          trace.Tracer
	repo            repository.Connection
	describe        describe.SchedulerServiceClient
	inventory       inventory.InventoryServiceClient
	masterAccessKey string
	masterSecretKey string
}

func NewConnection(
	repo repository.Connection,
	kms *vault.KMSVaultSourceConfig,
	keyARN string,
	describe describe.SchedulerServiceClient,
	inventory inventory.InventoryServiceClient,
	masterAccessKey string,
	masterSecretKey string,
) Connection {
	return Connection{
		tracer:          otel.GetTracerProvider().Tracer("integration.service.sources"),
		repo:            repo,
		keyARN:          keyARN,
		kms:             kms,
		inventory:       inventory,
		describe:        describe,
		masterAccessKey: masterAccessKey,
		masterSecretKey: masterSecretKey,
	}
}

func (h Connection) CredentialV2ToV1(newCred string) (string, error) {
	cnf, err := h.kms.Decrypt(newCred, h.keyARN)
	if err != nil {
		return "", err
	}

	awsCnf, err := entity.AWSCredentialV2ConfigFromMap(cnf)
	if err != nil {
		return "", err
	}

	newConf := entity.AWSCredentialConfig{
		AccountId:            awsCnf.AccountID,
		Regions:              nil,
		AccessKey:            h.masterAccessKey,
		SecretKey:            h.masterSecretKey,
		AssumeRoleName:       awsCnf.AssumeRoleName,
		AssumeAdminRoleName:  awsCnf.AssumeRoleName,
		AssumeRolePolicyName: "",
		ExternalId:           awsCnf.ExternalId,
	}
	newSecret, err := h.kms.Encrypt(newConf.AsMap(), h.keyARN)
	if err != nil {
		return "", err
	}

	return string(newSecret), nil
}

func (h Connection) Data(
	ctx *httpclient.Context,
	ids []string,
	resourceCollections []string,
	startTime, endTime *time.Time,
	needCost, needResourceCount bool,
) (map[string]inventoryAPI.ConnectionData, error) {
	connectionData, err := h.inventory.ListConnectionsData(ctx, nil, resourceCollections, startTime, endTime, needCost, needResourceCount)
	if err != nil {
		return nil, err
	}

	return connectionData, nil
}

func (h Connection) Pending(ctx *httpclient.Context) ([]string, error) {
	pending, err := h.describe.ListPendingConnections(ctx)
	if err != nil {
		return nil, err
	}

	return pending, nil
}

func (h Connection) List(ctx context.Context, types []source.Type) ([]model.Connection, error) {
	var (
		connections []model.Connection
		err         error
	)

	_, span := h.tracer.Start(ctx, "list", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	if len(types) > 0 {
		connections, err = h.repo.ListOfTypes(ctx, types)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.End()
	} else {
		connections, err = h.repo.List(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	return connections, nil
}

func (h Connection) Get(ctx context.Context, ids []string) ([]model.Connection, error) {
	_, span := h.tracer.Start(ctx, "get", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	connections, err := h.repo.Get(ctx, ids)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, err
	}

	return connections, nil
}

func (h Connection) Count(ctx context.Context, t *source.Type) (int64, error) {
	_, span := h.tracer.Start(ctx, "count", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	var (
		count int64
		err   error
	)

	if t == nil {
		count, err = h.repo.Count(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return 0, err
		}
	} else {
		count, err = h.repo.CountOfType(ctx, *t)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return 0, err
		}
	}

	return count, nil
}

func (h Connection) ListWithFilter(
	ctx context.Context,
	types []source.Type,
	ids []string,
	lifecycleState []model.ConnectionLifecycleState,
	healthStates []source.HealthStatus,
) ([]model.Connection, error) {
	_, span := h.tracer.Start(ctx, "count", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	h.repo.ListWithFilters(ctx, types, ids, lifecycleState, healthStates)

	return nil, nil
}
