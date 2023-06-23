package main

import (
	"os"

	compliance_report "github.com/kaytu-io/kaytu-engine/pkg/compliance"
)

func main() {
	if err := compliance_report.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
