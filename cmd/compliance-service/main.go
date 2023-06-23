package main

import (
	"fmt"
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/compliance"
)

func main() {
	if err := compliance.ServerCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
