package main

import (
	"os"

	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/spf13/cobra"
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
