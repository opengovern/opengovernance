package main

import (
	"os"

	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer"
)

func main() {
	if err := summarizer.WorkerCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
