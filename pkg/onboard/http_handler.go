package onboard

import (
	"fmt"

	"github.com/hashicorp/vault/api/auth/kubernetes"
	"go.uber.org/zap"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/queue"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
)

type HttpHandler struct {
	db                    Database
	sourceEventsQueue     queue.Interface
	vault                 vault.SourceConfig
	awsPermissionCheckURL string
}

func InitializeHttpHandler(
	rabbitMQUsername string,
	rabbitMQPassword string,
	rabbitMQHost string,
	rabbitMQPort int,
	sourceEventsQueueName string,
	postgresUsername string,
	postgresPassword string,
	postgresHost string,
	postgresPort string,
	postgresDb string,
	postgresSSLMode string,
	vaultAddress string,
	vaultToken string,
	vaultRoleName string,
	vaultCaPath string,
	vaultUseTLS bool,
	logger *zap.Logger,
	awsPermissionCheckURL string,
) (*HttpHandler, error) {

	fmt.Println("Initializing http handler")

	// setup source events queue
	qCfg := queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = sourceEventsQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = "onboard-service"
	sourceEventsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to the source queue: ", sourceEventsQueueName)

	cfg := postgres.Config{
		Host:    postgresHost,
		Port:    postgresPort,
		User:    postgresUsername,
		Passwd:  postgresPassword,
		DB:      postgresDb,
		SSLMode: postgresSSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	fmt.Println("Connected to the postgres database: ", postgresDb)

	k8sAuth, err := kubernetes.NewKubernetesAuth(
		vaultRoleName,
		kubernetes.WithServiceAccountToken(vaultToken),
	)
	if err != nil {
		return nil, err
	}

	// setup vault
	v, err := vault.NewSourceConfig(vaultAddress, vaultCaPath, k8sAuth, vaultUseTLS)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to vault:", vaultAddress)

	db := Database{orm: orm}
	err = db.Initialize()
	if err != nil {
		return nil, err
	}
	fmt.Println("Initialized postgres database: ", postgresDb)

	return &HttpHandler{
		vault:                 v,
		db:                    db,
		sourceEventsQueue:     sourceEventsQueue,
		awsPermissionCheckURL: awsPermissionCheckURL,
	}, nil
}
