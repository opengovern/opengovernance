package main

import (
	"github.com/kaytu-io/kaytu-engine/pkg/hopper"
	"os"
)

func main() {
	if err := hopper.HopperCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
