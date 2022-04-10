package main

import (
	"os"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth"
)

func main() {
	if err := auth.Command().Execute(); err != nil {
		os.Exit(1)
	}
}
