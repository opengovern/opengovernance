package onboard

import "github.com/spf13/cobra"

var Create = &cobra.Command{
	Use:   "create",
	Short: "this use for create user or key",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			err := cmd.Help()
			if err != nil {
				return err
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	Create.AddCommand()
}
