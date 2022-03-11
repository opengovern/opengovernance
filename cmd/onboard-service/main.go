package main

import (
	"fmt"
	"os"

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
	VaultToken    = os.Getenv("VAULT_TOKEN")
	VaultRoleName = os.Getenv("VAULT_ROLE")

	HttpAddress = os.Getenv("HTTP_ADDRESS")
)

func main() {
	r := onboard.InitializeRouter()
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

	h.Register(r)
	r.Logger.Fatal(r.Start(HttpAddress))
}
