package main

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/subscription"
	"os"
)

func main() {
	if err := subscription.Command().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
