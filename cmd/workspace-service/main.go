package main

import (
	"os"

	"gitlab.com/keibiengine/keibi-engine/pkg/workspace"
)

func main() {
	if err := workspace.Command().Execute(); err != nil {
		os.Exit(1)
	}
}
