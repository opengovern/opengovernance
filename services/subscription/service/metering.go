package service

import (
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	client2 "github.com/kaytu-io/kaytu-engine/pkg/auth/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	workspaceClient "github.com/kaytu-io/kaytu-engine/pkg/workspace/client"
	"github.com/kaytu-io/kaytu-engine/services/subscription/api/entities"
	"github.com/kaytu-io/kaytu-engine/services/subscription/config"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db"
	"go.uber.org/zap"
	"time"
)

type MeteringService struct {
	logger *zap.Logger
	db     db.Database
	cnf    config.SubscriptionConfig

	firehoseClient *firehose.Client

	workspaceClient workspaceClient.WorkspaceServiceClient
	authClient      client2.AuthServiceClient
}

func NewMeteringService(
	logger *zap.Logger,
	db db.Database,
	cnf config.SubscriptionConfig,
	firehoseClient *firehose.Client,
	workspaceClient workspaceClient.WorkspaceServiceClient,
	authClient client2.AuthServiceClient,
) MeteringService {
	return MeteringService{
		logger:          logger.Named("meteringSvc"),
		db:              db,
		cnf:             cnf,
		firehoseClient:  firehoseClient,
		workspaceClient: workspaceClient,
		authClient:      authClient,
	}
}

func (svc MeteringService) GetMeters(userID string, startTime, endTime time.Time) ([]entities.Meter, error) {
	workspaces, err := svc.workspaceClient.ListWorkspaces(&httpclient.Context{UserRole: api.InternalRole, UserID: userID})
	if err != nil {
		return nil, err
	}

	meterTypes := entities.ListAllMeterTypes()

	var meters []entities.Meter
	for _, meterType := range meterTypes {
		for _, workspace := range workspaces {
			if workspace.OwnerId == nil || *workspace.OwnerId != userID {
				continue
			}

			var meterValue float64
			if meterType.IsTotal() {
				value, err := svc.db.AvgOfMeter([]string{workspace.ID}, meterType, startTime, endTime)
				if err != nil {
					return nil, err
				}
				meterValue = value
			} else {
				value, err := svc.db.SumOfMeter([]string{workspace.ID}, meterType, startTime, endTime)
				if err != nil {
					return nil, err
				}
				meterValue = float64(value)
			}

			meters = append(meters, entities.Meter{
				WorkspaceName: workspace.Name,
				Type:          meterType,
				IsTotal:       meterType.IsTotal(),
				Value:         meterValue,
			})
		}
	}

	return meters, nil
}

func (svc MeteringService) RunChecks() {
	// get list of workspaces.
	workspaces, err := svc.workspaceClient.ListWorkspaces(&httpclient.Context{UserRole: api.InternalRole, UserID: api.GodUserID})
	if err != nil {
		svc.logger.Error("failed to list workspaces", zap.Error(err))
		return
	}

	svc.logger.Info("listing workspaces done", zap.Int("count", len(workspaces)))
	for _, ws := range workspaces {
		if !ws.IsCreated {
			continue
		}

		if ws.Status != api2.StateID_Provisioned &&
			ws.Status != api2.StateID_Provisioning &&
			ws.Status != api2.StateID_WaitingForCredential {
			continue
		}

		tim := time.Now().Add(-1 * time.Hour)
		previousHour := time.Date(tim.Year(), tim.Month(), tim.Day(), tim.Hour(), 0, 0, 0, tim.Location())

		meterTypes := entities.ListAllMeterTypes()
		for _, meterType := range meterTypes {
			svc.logger.Info("checking meter", zap.String("workspaceID", ws.ID), zap.String("hour", previousHour.String()), zap.String("meter", string(meterType)))
			meter, err := svc.db.GetMeter(ws.ID, previousHour, meterType)
			if err != nil {
				svc.logger.Error("failed to get meter", zap.Error(err), zap.String("workspaceID", ws.ID), zap.String("meter", string(meterType)))
				return
			}

			if meter == nil {
				svc.logger.Info("generating metric", zap.String("workspaceID", ws.ID), zap.String("meter", string(meterType)))
				err = svc.generateMeter(ws.ID, previousHour, meterType)
				if err != nil {
					svc.logger.Error("failed to generate meter",
						zap.Error(err),
						zap.String("workspaceID", ws.ID),
						zap.String("hour", previousHour.String()),
						zap.String("meter", string(meterType)),
					)
					return
				}
			} else {
				svc.logger.Info("metrics is already there", zap.Int64("value", meter.Value))
			}
		}
	}
}
