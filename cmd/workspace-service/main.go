package main

import (
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/workspace"
)

func main() {
	if err := workspace.Command().Execute(); err != nil {
		os.Exit(1)
	}
}
