package main

import (
	"fmt"
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/checkup"
)

func main() {
	if err := checkup.WorkerCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
