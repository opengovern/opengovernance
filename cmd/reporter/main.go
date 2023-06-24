package main

import (
	"github.com/kaytu-io/kaytu-engine/pkg/reporter"
	"os"
)

func main() {
	if err := reporter.ReporterCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
