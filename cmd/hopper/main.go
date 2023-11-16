package main

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/hopper"
	"os"
)

func main() {
	if err := hopper.HopperCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
