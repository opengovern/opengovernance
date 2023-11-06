package main

import (
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"os"
)

func main() {
	if err := runner.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
