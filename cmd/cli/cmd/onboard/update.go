package onboard

import "github.com/spf13/cobra"

var Update = &cobra.Command{
	Use:   "update",
	Short: "it is use for update user or key user  ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
}

func init() {
	Update.AddCommand()
}
