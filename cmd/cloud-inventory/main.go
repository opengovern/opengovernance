package main

import (
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
		os.Exit(1)
	}
}
