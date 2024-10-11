package auth

import (
	"context"
	"encoding/json"
	"fmt"
	dexApi "github.com/dexidp/dex/api/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/open-governance/pkg/auth/auth0"
	"github.com/kaytu-io/open-governance/pkg/auth/db"
	"github.com/kaytu-io/open-governance/services/migrator/config"
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

	wm, err := dbm.GetWorkspaceMapByName("main")
	if err != nil {
		logger.Error("Auth Migrator: failed to get workspace", zap.Error(err))
		return err
	}

	role := api.AdminRole

	var appMetadata auth0.Metadata
	appMetadata.WorkspaceAccess = map[string]api.Role{
		wm.ID: role,
	}
	appMetadataJson, err := json.Marshal(appMetadata)
	if err != nil {
		logger.Error("Auth Migrator: failed to marshal app metadata json", zap.Error(err))
		return err
	}

	appMetadataJsonb := pgtype.JSONB{}
	err = appMetadataJsonb.Set(appMetadataJson)
	if err != nil {
		logger.Error("Auth Migrator: failed to set app metadata json", zap.Error(err))
		return err
	}

	userMetadataJsonb := pgtype.JSONB{}
	err = userMetadataJsonb.Set([]byte(""))
	if err != nil {
		return err
	}

	user := &db.User{
		UserUuid:     uuid.New(),
		Email:        conf.DefaultDexUserEmail,
		Username:     conf.DefaultDexUserEmail,
		Name:         conf.DefaultDexUserEmail,
		IdLifecycle:  db.UserLifecycleActive,
		Role:         role,
		UserId:       fmt.Sprintf("dex|%s", conf.DefaultDexUserEmail),
		AppMetadata:  appMetadataJsonb,
		UserMetadata: userMetadataJsonb,
		StaticOwner:  true,
		Connector:    "local",
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
