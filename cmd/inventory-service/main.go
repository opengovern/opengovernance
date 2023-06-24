package main

import (
	"fmt"
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/inventory"
)

func main() {
	if err := inventory.Command().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
