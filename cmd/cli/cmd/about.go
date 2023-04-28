package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

// aboutCmd represents the about command
var aboutCmd = &cobra.Command{
	Use:   "about",
	Short: "About user",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags {
			fmt.Println("please enter right flag .")
			return cmd.Help()
		}
		if len(args) == 0 {
			fmt.Println(cmd.Help())
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := cli.GetConfig()
		if err != nil {
			return fmt.Errorf("[about]: %v", err)
		}
		checkEXP, err := cli.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("your access token was expire please login again ")
			return nil
		}

		bodyResponse, err := cli.RequestAbout(accessToken)
		if err != nil {
			return fmt.Errorf("[about]: %v", err)
		}
		err = cli.PrintOutput(bodyResponse, OutputType)
		if err != nil {
			return fmt.Errorf("[about]: %v", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(aboutCmd)
	aboutCmd.Flags().StringVar(&OutputType, "output", "", "specifying output type [json, table]")
}
