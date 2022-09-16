package main

import (
	"fmt"
	"os"

	"gitlab.com/keibiengine/keibi-engine/pkg/compliance"
)

func main() {
	if err := compliance.ServerCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
