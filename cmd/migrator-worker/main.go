package main

import (
	"fmt"
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/migrator"
)

func main() {
	if err := migrator.JobCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
