package main

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/hopper"
	"os"
)

func main() {
	if err := hopper.HopperCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
