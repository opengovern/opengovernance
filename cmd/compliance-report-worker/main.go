package main

import (
	compliance_report "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
	"os"
)

func main() {
	if err := compliance_report.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
