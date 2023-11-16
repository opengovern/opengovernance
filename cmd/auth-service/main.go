package main

import (
	"fmt"
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/auth"
)

func main() {
	if err := auth.Command().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
