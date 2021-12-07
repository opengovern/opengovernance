package main

import (
	"fmt"
	"os"

	swagger "github.com/swaggo/echo-swagger"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard"
)

const (
	SourceEventsQueueName = "source-events-queue"
)

var (
	RabbitMQService  = os.Getenv("RABBITMQ_SERVICE")
	RabbitMQPort     = 5672
	RabbitMQUsername = os.Getenv("RABBITMQ_USERNAME")
	RabbitMQPassword = os.Getenv("RABBITMQ_PASSWORD")

	PostgreSQLHost     = os.Getenv("POSTGRESQL_HOST")
	PostgreSQLPort     = os.Getenv("POSTGRESQL_PORT")
	PostgreSQLDb       = os.Getenv("POSTGRESQL_DB")
	PostgreSQLUser     = os.Getenv("POSTGRESQL_USERNAME")
	PostgreSQLPassword = os.Getenv("POSTGRESQL_PASSWORD")

	VaultAddress  = os.Getenv("VAULT_ADDRESS")
	VaultToken    = os.Getenv("VAULT_AUTH_JWT")
	VaultRoleName = os.Getenv("VAULT_ONBOARD_ROLE")
)

func main() {
	r := onboard.InitializeRouter()
	r.GET("/swagger/*", swagger.WrapHandler)

	// TODO: http handler shouldn't be initializing the queue & the db.
	h, err := onboard.InitializeHttpHandler(
		RabbitMQUsername,
		RabbitMQPassword,
		RabbitMQService,
		RabbitMQPort,
		SourceEventsQueueName,
		PostgreSQLUser,
		PostgreSQLPassword,
		PostgreSQLHost,
		PostgreSQLPort,
		PostgreSQLDb,
		VaultAddress,
		VaultToken,
		VaultRoleName,
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	h.Register(r.Group("/api/v1"))
	r.Logger.Fatal(r.Start("127.0.0.1:6251"))
}
