package main

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"os"
)

func main() {
	if err := runner.WorkerCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
