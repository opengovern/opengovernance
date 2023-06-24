package main

import (
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/describe"
)

func main() {
	if err := describe.SchedulerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
