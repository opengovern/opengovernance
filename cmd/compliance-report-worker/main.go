package main

import (
	"os"

	compliance_report "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
)

func main() {
	if err := compliance_report.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
