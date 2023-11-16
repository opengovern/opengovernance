package main

import (
	"fmt"
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/workspace"
)

func main() {
	if err := workspace.Command().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
