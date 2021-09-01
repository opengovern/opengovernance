package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/anil94/golang-aws-inventory/pkg/aws"
)

func main() {
	rootCmd := cobra.Command{
		Use: "cloud-inventory",
	}

	rootCmd.AddCommand(
		aws.Command(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
