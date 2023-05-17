package hopper

import (
	"github.com/spf13/cobra"
)

func HopperCommand() *cobra.Command {
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			h := HttpServer{}
			return h.Run()
		},
	}

	return cmd
}
