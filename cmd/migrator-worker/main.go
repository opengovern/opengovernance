package main

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/migrator"
	"os"
)

func main() {
	if err := migrator.Command().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
