package main

import (
	"fmt"
	"os"

	swagger "github.com/swaggo/echo-swagger"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory"
	_ "gitlab.com/keibiengine/keibi-engine/pkg/inventory/docs" // docs is generated by Swag CLI, you have to import it.
)

var (
	ElasticSearchAddress  = os.Getenv("ES_ADDRESS")
	ElasticSearchUsername = os.Getenv("ES_USERNAME")
	ElasticSearchPassword = os.Getenv("ES_PASSWORD")

	HttpAddress = os.Getenv("HTTP_ADDRESS")
)

// @title Inventory Service API
// @version 1.0
// @description Inventory service

// @host https://dev-cluster.keibi.io
// @BasePath /inventory/api/v1
func main() {
	r := inventory.InitializeRouter()
	r.GET("/swagger/*", swagger.WrapHandler)

	h, err := inventory.InitializeHttpHandler(
		ElasticSearchAddress,
		ElasticSearchUsername,
		ElasticSearchPassword,
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	h.Register(r.Group("/api/v1"))
	r.Logger.Fatal(r.Start(HttpAddress))
}
