package main

import (
	"fmt"
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/describe"
)

func main() {
	if err := describe.SchedulerCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
