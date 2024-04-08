package workspace

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := koanf.Provide("workspace", config.Config{})

			s, err := NewServer(cmd.Context(), cfg)
			if err != nil {
				return fmt.Errorf("new server: %w", err)
			}
			return s.Start(cmd.Context())
		},
	}
	return cmd
}
