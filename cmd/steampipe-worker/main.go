package main

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"
	"os"
)

func main() {
	if err := steampipe.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
