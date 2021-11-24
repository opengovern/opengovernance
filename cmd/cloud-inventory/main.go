package main

import (
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
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
