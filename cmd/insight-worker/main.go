package main

import (
	"fmt"
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/insight"
)

func main() {
	if err := insight.WorkerCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
