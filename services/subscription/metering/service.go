package metering

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	client2 "github.com/kaytu-io/kaytu-engine/pkg/auth/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/client"
	"github.com/kaytu-io/kaytu-engine/services/subscription/config"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db/model"
	"go.uber.org/zap"
	"time"
)

type Service struct {
	pdb    db.Database
	logger *zap.Logger
	cnf    config.SubscriptionConfig

	workspaceClient client.WorkspaceServiceClient
	authClient      client2.AuthServiceClient
}

func New(logger *zap.Logger, cnf config.SubscriptionConfig, pdb db.Database) (*Service, error) {
	workspaceClient := client.NewWorkspaceClient(cnf.Workspace.BaseURL)
	authClient := client2.NewAuthServiceClient(cnf.Auth.BaseURL)

	return &Service{
		logger:          logger,
		pdb:             pdb,
		cnf:             cnf,
		workspaceClient: workspaceClient,
		authClient:      authClient,
	}, nil
}

func (s *Service) Run() {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("metering paniced", zap.Error(fmt.Errorf("%v", r)))
			time.Sleep(5 * time.Second)
			go s.Run()
		}
	}()

	fmt.Println(s.cnf)

	for {
		s.logger.Info("starting checks")
		s.runChecks()
		time.Sleep(10 * time.Minute)
	}
}

func (s *Service) runChecks() {
	// get list of workspaces.
	workspaces, err := s.workspaceClient.ListWorkspaces(&httpclient.Context{UserRole: api.InternalRole, UserID: api.GodUserID})
	if err != nil {
		s.logger.Error("failed to list workspaces", zap.Error(err))
		return
	}

	s.logger.Info("listing workspaces done", zap.Int("count", len(workspaces)))
	for _, ws := range workspaces {
		if !ws.IsCreated {
			continue
		}

		if ws.Status != api2.StatusProvisioned && ws.Status != api2.StatusBootstrapping {
			continue
		}

		previousHour := time.Now().Add(-1 * time.Hour).Format("2006-01-02-15")
		meterTypes := model.ListAllMeterTypes()
		for _, meterType := range meterTypes {
			s.logger.Info("checking meter", zap.String("workspaceID", ws.ID), zap.String("meter", string(meterType)))
			meter, err := s.pdb.GetMeter(ws.ID, previousHour, meterType)
			if err != nil {
				s.logger.Error("failed to get meter", zap.Error(err), zap.String("workspaceID", ws.ID), zap.String("meter", string(meterType)))
				return
			}

			if meter == nil {
				s.logger.Info("generating metric", zap.String("workspaceID", ws.ID), zap.String("meter", string(meterType)))
				err = s.generateMeter(ws.ID, previousHour, meterType)
				if err != nil {
					s.logger.Error("failed to generate meter",
						zap.Error(err),
						zap.String("workspaceID", ws.ID),
						zap.String("hour", previousHour),
						zap.String("meter", string(meterType)),
					)
					return
				}
			} else {
				s.logger.Info("metrics is already there", zap.Int64("value", meter.Value))
			}
		}
	}
}
