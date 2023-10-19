package main

import (
	"fmt"
	"os"

	cost "github.com/kaytu-io/kaytu-engine/pkg/cost-estimator"
)

func main() {
	if err := cost.Command().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
