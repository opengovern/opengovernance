package main

import (
	"os"

	"gitlab.com/keibiengine/keibi-engine/pkg/checkup"
)

func main() {
	if err := checkup.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
