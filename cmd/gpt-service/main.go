package main

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/gpt"
	"os"
)

func main() {
	if err := gpt.Command().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
