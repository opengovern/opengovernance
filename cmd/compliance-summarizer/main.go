package main

import (
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer"
	"os"
)

func main() {
	if err := summarizer.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
