package main

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics"
	"os"
)

func main() {
	if err := analytics.WorkerCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
