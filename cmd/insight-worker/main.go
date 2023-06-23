package main

import (
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/insight"
)

func main() {
	if err := insight.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
