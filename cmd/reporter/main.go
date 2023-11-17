package main

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/reporter"
	"os"
)

func main() {
	if err := reporter.ReporterCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
