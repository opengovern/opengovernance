package workspace

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	config2 "github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg config.Config
			config2.ReadFromEnv(&cfg, nil)

			s, err := NewServer(cmd.Context(), cfg)
			if err != nil {
				return fmt.Errorf("new server: %w", err)
			}
			return s.Start(cmd.Context())
		},
	}
	return cmd
}
