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

	HttpAddress = os.Getenv("HTTP_ADDRESS")
)

func main() {
	r := inventory.InitializeRouter()

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
