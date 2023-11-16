package main

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer"
	"os"
)

func main() {
	if err := summarizer.WorkerCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
