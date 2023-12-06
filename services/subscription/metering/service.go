package metering

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	client2 "github.com/kaytu-io/kaytu-engine/pkg/auth/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
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

func New(logger *zap.Logger, cnf config.SubscriptionConfig) (*Service, error) {
	pdb, err := db.NewDatabase(cnf.Postgres, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}

	workspaceClient := client.NewWorkspaceClient(cnf.Workspace.BaseURL)
	authClient := client2.NewAuthServiceClient(cnf.Auth.BaseURL)

	return &Service{
		logger:          logger,
		pdb:             pdb,
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

	for {
		s.runChecks()
		time.Sleep(10 * time.Minute)
	}
}

func (s *Service) runChecks() {
	// get list of workspaces.
	workspaces, err := s.workspaceClient.ListWorkspaces(&httpclient.Context{UserRole: api.InternalRole})
	if err != nil {
		s.logger.Error("failed to list workspaces checks", zap.Error(err))
		return
	}

	for _, ws := range workspaces {
		previousHour := time.Now().Add(-1 * time.Hour).Format("2006-01-02-15")
		meterTypes := model.ListAllMeterTypes()
		for _, meterType := range meterTypes {
			meter, err := s.pdb.GetMeter(ws.ID, previousHour, meterType)
			if err != nil {
				s.logger.Error("failed to get meter", zap.Error(err), zap.String("workspaceID", ws.ID), zap.String("meter", string(meterType)))
				return
			}

			if meter == nil {
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
			}
		}
	}
}
