package main

import (
	"fmt"
	"os"

	"gitlab.com/keibiengine/keibi-engine/pkg/migrator"
)

func main() {
	if err := migrator.JobCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
