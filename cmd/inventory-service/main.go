package main

import (
	"fmt"
	"os"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory"
)

func main() {
	if err := inventory.Command().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
