package main

import (
	"fmt"
	"os"

	"github.com/kaytu-io/kaytu-engine/services/integration"
)

func main() {
	if err := integration.Command().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
