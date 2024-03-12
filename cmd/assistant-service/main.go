package main

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/assistant"
	"os"
)

func main() {
	if err := assistant.Command().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
