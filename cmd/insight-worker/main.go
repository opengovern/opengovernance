package main

import (
	"os"

	"gitlab.com/keibiengine/keibi-engine/pkg/insight"
)

func main() {
	if err := insight.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
