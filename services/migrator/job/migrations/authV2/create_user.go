package authV2

import (
	"context"
	"fmt"
	dexApi "github.com/dexidp/dex/api/v2"
	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/postgres"
	"github.com/opengovern/opengovernance/pkg/authV2/db"
	"github.com/opengovern/opengovernance/services/migrator/config"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
)

type Migration struct{}

func (m Migration) IsGitBased() bool {
	return false
}

func (m Migration) AttachmentFolderPath() string {
	return ""
}

func (m Migration) Run(ctx context.Context, conf config.MigratorConfig, logger *zap.Logger) error {
	orm, err := postgres.NewClient(&postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      "auth",
		SSLMode: conf.PostgreSQL.SSLMode,
	}, logger)
	if err != nil {
		return fmt.Errorf("new postgres client: %w", err)
	}
	dbm := db.Database{Orm: orm}

	count, err := dbm.GetUsersCount()
	if err != nil {
		return err
	}
	if count > 0 {
		logger.Warn("users already exist")
		return nil
	}

	dexClient, err := newDexClient(conf.DexGrpcAddress)
	if err != nil {
		logger.Error("Auth Migrator: failed to create dex client", zap.Error(err))
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(conf.DefaultDexUserPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Auth Migrator: failed to generate password", zap.Error(err))
		return err
	}

	publicClientReq := dexApi.CreateClientReq{
		Client: &dexApi.Client{
			Id:   "public-client",
			Name: "Public Client",
			RedirectUris: []string{
				"https://DOMAIN_NAME_PLACEHOLDER_DO_NOT_CHANGE/callback",
				"http://DOMAIN_NAME_PLACEHOLDER_DO_NOT_CHANGE/callback",
				"http://localhost:3000/callback",
				"http://localhost:8080/callback",
			},
			Public: true,
		},
	}

	_, err = dexClient.CreateClient(ctx, &publicClientReq)
	if err != nil {
		logger.Error("Auth Migrator: failed to create dex public client", zap.Error(err))
		return err
	}

	privateClientReq := dexApi.CreateClientReq{
		Client: &dexApi.Client{
			Id:   "private-client",
			Name: "Private Client",
			RedirectUris: []string{
				"https://DOMAIN_NAME_PLACEHOLDER_DO_NOT_CHANGE/callback",
			},
			Secret: "secret",
		},
	}

	_, err = dexClient.CreateClient(ctx, &privateClientReq)
	if err != nil {
		logger.Error("Auth Migrator: failed to create dex private client", zap.Error(err))
		return err
	}

	req := dexApi.CreatePasswordReq{
		Password: &dexApi.Password{
			Email:    conf.DefaultDexUserEmail,
			Username: conf.DefaultDexUserName,
			UserId:   fmt.Sprintf("dex|%s", conf.DefaultDexUserEmail),
			Hash:     hashedPassword,
		},
	}

	_, err = dexClient.CreatePassword(ctx, &req)
	if err != nil {
		logger.Error("Auth Migrator: failed to create dex password", zap.Error(err))
		return err
	}

	role := api.AdminRole

	user := &db.User{

		Email:    conf.DefaultDexUserEmail,
		Username: conf.DefaultDexUserEmail,
		FullName: conf.DefaultDexUserEmail,

		Role:                  role,
		ExternalId:            fmt.Sprintf("dex|%s", conf.DefaultDexUserEmail),
		Connector:             "local",
		IsActive:              true,
		RequirePasswordChange: true,
	}
	err = dbm.CreateUser(user)
	if err != nil {
		logger.Error("Auth Migrator: failed to create user in database", zap.Error(err))
		return err
	}

	return nil
}

func newDexClient(hostAndPort string) (dexApi.DexClient, error) {
	conn, err := grpc.NewClient(hostAndPort, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("dial: %v", err)
	}
	return dexApi.NewDexClient(conn), nil
}
