package main

import (
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/checkup"
)

func main() {
	if err := checkup.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
