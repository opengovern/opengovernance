package main

import (
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/summarizer"
)

func main() {
	if err := summarizer.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
