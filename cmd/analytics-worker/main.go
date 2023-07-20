package main

import (
	"github.com/kaytu-io/kaytu-engine/pkg/analytics"
	"os"
)

func main() {
	if err := analytics.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
