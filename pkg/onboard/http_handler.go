package onboard

import (
	"fmt"

	"github.com/hashicorp/vault/api/auth/kubernetes"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type HttpHandler struct {
	db                *Database
	sourceEventsQueue *Queue
	vault             vault.Keibi
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
	vaultAddress string,
	vaultToken string,
	vaultRoleName string,
) (h *HttpHandler, err error) {

	h = &HttpHandler{}

	fmt.Println("Initializing http handler")

	// setup source events queue
	qCfg := QueueConfig{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = sourceEventsQueueName
	qCfg.Queue.Durable = true
	sourceEventsQueue, err := NewQueue(qCfg)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to the source queue: ", sourceEventsQueueName)
	h.sourceEventsQueue = sourceEventsQueue

	// setup postgres connection
	dsn := fmt.Sprintf(`host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=GMT`,
		postgresHost,
		postgresPort,
		postgresUsername,
		postgresPassword,
		postgresDb,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to the postgres database: ", postgresDb)
	h.db = &Database{orm: db}
	h.db.orm.AutoMigrate(
		&Organization{},
		&Source{},
	)

	k8sAuth, err := kubernetes.NewKubernetesAuth(
		vaultRoleName,
		kubernetes.WithServiceAccountToken(vaultToken),
	)
	if err != nil {
		return nil, err
	}

	// setup vault
	v, err := vault.NewVault(vaultAddress, k8sAuth)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to vault:", vaultAddress)
	h.vault = v

	return h, nil
}
