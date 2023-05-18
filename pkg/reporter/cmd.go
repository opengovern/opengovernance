package reporter

import (
	config2 "github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/spf13/cobra"
)

func ReporterCommand() *cobra.Command {
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			config := JobConfig{}
			
			config2.ReadFromEnv(&config, nil)

			j, err := New(config)
			if err != nil {
				return err
			}
			return j.Run()
		},
	}

	return cmd
}
