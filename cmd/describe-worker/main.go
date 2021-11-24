package main

import (
	"os"

	"gitlab.com/anil94/golang-aws-inventory/pkg/describe"
)

func main() {
	if err := describe.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
