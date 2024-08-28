package service

import (
	"context"
	"errors"
	"github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"time"

	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	inventoryAPI "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	inventory "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
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

var (
	ErrMaxConnectionsExceeded   = errors.New("number of connections exceeded")
	ErrInvalidMaxConnectionType = errors.New("max connections should be int or int64")
)

type Connection struct {
	keyId             string
	vault             vault.VaultSourceConfig
	tracer            trace.Tracer
	repo              repository.Connection
	transactionalRepo repository.CredConn
	describe          describe.SchedulerServiceClient
	inventory         inventory.InventoryServiceClient
	meta              *meta.Meta
	masterAccessKey   string
	masterSecretKey   string
	logger            *zap.Logger
}

func NewConnection(
	repo repository.Connection,
	transactionalRepo repository.CredConn,
	vault vault.VaultSourceConfig,
	keyARN string,
	describe describe.SchedulerServiceClient,
	inventory inventory.InventoryServiceClient,
	meta *meta.Meta,
	masterAccessKey string,
	masterSecretKey string,
	logger *zap.Logger,
) Connection {
	return Connection{
		tracer:            otel.GetTracerProvider().Tracer("integration.service.connection"),
		repo:              repo,
		transactionalRepo: transactionalRepo,
		keyId:             keyARN,
		vault:             vault,
		inventory:         inventory,
		describe:          describe,
		meta:              meta,
		masterAccessKey:   masterAccessKey,
		masterSecretKey:   masterSecretKey,
		logger:            logger.Named("service").Named("connection"),
	}
}

func (h Connection) Data(
	ctx *httpclient.Context,
	ids []string,
	resourceCollections []string,
	startTime, endTime *time.Time,
	needCost, needResourceCount bool,
) (map[string]inventoryAPI.ConnectionData, error) {
	connectionData, err := h.inventory.ListConnectionsData(ctx, nil, resourceCollections, startTime, endTime, nil, needCost, needResourceCount)
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

// List lists connection with given type, passing empty list of types means you don't want any kind of type filtering.
func (h Connection) List(ctx context.Context, types []source.Type) ([]model.Connection, error) {
	var (
		connections []model.Connection
		err         error
	)

	ctx, span := h.tracer.Start(ctx, "list")
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
	ctx, span := h.tracer.Start(ctx, "get")
	defer span.End()

	connections, err := h.repo.Get(ctx, ids)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, err
	}

	return connections, nil
}

func (h Connection) Count(ctx context.Context, t *source.Type, ct []model.CredentialType) (int64, error) {
	ctx, span := h.tracer.Start(ctx, "count")
	defer span.End()

	var (
		count int64
		err   error
	)

	if t == nil && ct == nil {
		count, err = h.repo.Count(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return 0, err
		}
	} else {
		count, err = h.repo.CountOfType(ctx, t, ct)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return 0, err
		}
	}

	return count, nil
}

// ListWithFilter lists connections based on the given filters. Filters are applied as
// where clause in the database query.
func (h Connection) ListWithFilter(
	ctx context.Context,
	types []source.Type,
	credTypes []model.CredentialType,
	ids []string,
	lifecycleState []model.ConnectionLifecycleState,
	healthStates []source.HealthStatus,
) ([]model.Connection, error) {
	ctx, span := h.tracer.Start(ctx, "list-with-filter")
	defer span.End()

	conns, err := h.repo.ListWithFilters(ctx, types, credTypes, ids, lifecycleState, healthStates)
	if err != nil {
		return nil, err
	}

	return conns, nil
}

// Create a new connection in the database based on the given instance.
func (h Connection) Create(
	ctx context.Context,
	c model.Connection,
) error {
	ctx, span := h.tracer.Start(ctx, "create")
	defer span.End()

	return h.repo.Create(ctx, c)
}

// Update updates given connection in the database.
func (h Connection) Update(
	ctx context.Context,
	c model.Connection,
) error {
	ctx, span := h.tracer.Start(ctx, "update")
	defer span.End()

	return h.repo.Update(ctx, c)
}

// MaxConnections reads the maximum number of the available connection in the workspace
// from the metadata service.
func (h Connection) MaxConnections(ctx context.Context) (int64, error) {
	ctx, span := h.tracer.Start(ctx, "count-by-credential")
	defer span.End()

	cnf, err := h.meta.Client.GetConfigMetadata(&httpclient.Context{UserRole: api.InternalRole}, models.MetadataKeyConnectionLimit)
	if err != nil {
		return 0, err
	}

	var maxConnections int64

	switch v := cnf.GetValue().(type) {
	case int64:
		maxConnections = v
	case int:
		maxConnections = int64(v)
	default:
		return 0, ErrInvalidMaxConnectionType
	}

	return maxConnections, nil
}

// UpdateHealth update the health status of the connection. using update database flag,
// you can control the database record should be updated or not.
func (h Connection) UpdateHealth(
	ctx context.Context,
	connection model.Connection,
	healthStatus source.HealthStatus,
	reason *string,
	spendDiscovery, assetDiscovery *bool,
	updateDatabase bool,
) (model.Connection, error) {
	connection.HealthState = healthStatus
	connection.HealthReason = reason
	connection.LastHealthCheckTime = time.Now()
	connection.SpendDiscovery = spendDiscovery
	connection.AssetDiscovery = assetDiscovery

	if updateDatabase == true {
		ctx, span := h.tracer.Start(ctx, "update-health")
		defer span.End()

		if err := h.repo.Update(ctx, connection); err != nil {
			return model.Connection{}, err
		}
	}

	return connection, nil
}

func (h Connection) CountByCredential(ctx context.Context, credentialID string, states []model.ConnectionLifecycleState, healthStates []source.HealthStatus) (int64, error) {
	ctx, span := h.tracer.Start(ctx, "count-by-credential")
	defer span.End()

	count, err := h.repo.CountByCredential(ctx, credentialID, states, healthStates)
	if err != nil {
		return 0, err
	}

	return count, err
}

func (h Connection) Delete(ctx context.Context, conn model.Connection) error {
	ctx, span := h.tracer.Start(ctx, "delete")
	defer span.End()

	if err := h.transactionalRepo.DeleteConnection(ctx, conn); err != nil {
		return err
	}

	return nil
}

func (h Connection) ListByCredential(ctx context.Context, credentialID string) ([]model.Connection, error) {
	ctx, span := h.tracer.Start(ctx, "delete")
	defer span.End()

	conns, err := h.repo.ListByCredential(ctx, credentialID)
	if err != nil {
		return nil, err
	}

	return conns, nil
}
