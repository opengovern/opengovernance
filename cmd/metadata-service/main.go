package main

import (
	"fmt"
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/metadata"
)

func main() {
	if err := metadata.Command().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
