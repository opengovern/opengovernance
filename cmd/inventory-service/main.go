package main

import (
	"fmt"
	"os"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory"
)

var (
	ElasticSearchAddress  = os.Getenv("ES_ADDRESS")
	ElasticSearchUsername = os.Getenv("ES_USERNAME")
	ElasticSearchPassword = os.Getenv("ES_PASSWORD")

	PostgreSQLHost     = os.Getenv("POSTGRESQL_HOST")
	PostgreSQLPort     = os.Getenv("POSTGRESQL_PORT")
	PostgreSQLDb       = os.Getenv("POSTGRESQL_DB")
	PostgreSQLUser     = os.Getenv("POSTGRESQL_USERNAME")
	PostgreSQLPassword = os.Getenv("POSTGRESQL_PASSWORD")

	SteampipeHost     = os.Getenv("STEAMPIPE_HOST")
	SteampipePort     = os.Getenv("STEAMPIPE_PORT")
	SteampipeDb       = os.Getenv("STEAMPIPE_DB")
	SteampipeUser     = os.Getenv("STEAMPIPE_USERNAME")
	SteampipePassword = os.Getenv("STEAMPIPE_PASSWORD")

	SchedulerBaseUrl = os.Getenv("SCHEDULER_BASE_URL")

	HttpAddress = os.Getenv("HTTP_ADDRESS")
)

func main() {
	handler, err := inventory.InitializeHttpHandler(
		ElasticSearchAddress,
		ElasticSearchUsername,
		ElasticSearchPassword,
		PostgreSQLHost,
		PostgreSQLPort,
		PostgreSQLDb,
		PostgreSQLUser,
		PostgreSQLPassword,
		SteampipeHost,
		SteampipePort,
		SteampipeDb,
		SteampipeUser,
		SteampipePassword,
		SchedulerBaseUrl,
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	r, err := inventory.InitializeRouter(handler)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	r.Logger.Fatal(r.Start(HttpAddress))
}
