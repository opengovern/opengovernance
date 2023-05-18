package main

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/reporter"
	"os"
)

func main() {
	if err := reporter.ReporterCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
