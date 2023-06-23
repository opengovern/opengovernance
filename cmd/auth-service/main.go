package main

import (
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/auth"
)

func main() {
	if err := auth.Command().Execute(); err != nil {
		os.Exit(1)
	}
}
