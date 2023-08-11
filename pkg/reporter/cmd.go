package reporter

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/config"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	internal "github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	config2 "github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

func ReporterCommand() *cobra.Command {
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			config := JobConfig{}
			config2.ReadFromEnv(&config, nil)
			j, err := New(config)
			if err != nil {
				panic(err)
			}

			j.Run()
			return startHttpServer(cmd.Context())
		},
	}

	return cmd
}

type ServerConfig struct {
	Http config.HttpServer
}

func startHttpServer(ctx context.Context) error {
	var conf ServerConfig
	config2.ReadFromEnv(&conf, nil)

	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}

	var handler internal.Routes
	if err != nil {
		return fmt.Errorf("init http handler: %w", err)
	}

	return httpserver.RegisterAndStart(logger, conf.Http.Address, handler)
}
