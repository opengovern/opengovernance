package main

import (
	"os"

	"gitlab.com/anil94/golang-aws-inventory/pkg/tasks"
)

func main() {
	if err := tasks.ConsumeCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
