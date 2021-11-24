package main

import (
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/anil94/golang-aws-inventory/pkg/aws"
	"gitlab.com/anil94/golang-aws-inventory/pkg/azure"
)

func main() {
	rootCmd := cobra.Command{
		Use: "cloud-inventory",
	}

	rootCmd.AddCommand(
		aws.Command(),
		azure.Command(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
