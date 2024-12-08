package tasks

import (
	"context"
	"fmt"
	"os"

	config2 "github.com/opengovern/og-util/pkg/config"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/postgres"
	"github.com/opengovern/opencomply/services/tasks/db"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	
	httpServerAddress     = os.Getenv("HTTP_ADDRESS")

	
)

func Command() *cobra.Command {
	return &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			return start(cmd.Context())
		},
	}
}

type ServerConfig struct {
	PostgreSQL config2.Postgres
}

// start runs both HTTP and GRPC server.
// GRPC server has Check method to ensure user is
// authenticated and authorized to perform an action.
// HTTP server has multiple endpoints to view and update
// the user roles.
func start(ctx context.Context) error {
	var conf ServerConfig
	config2.ReadFromEnv(&conf, nil)

	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}

	logger = logger.Named("tasks")

	//m := email.NewSendGridClient(mailApiKey, mailSender, mailSenderName, logger)

	// setup postgres connection
	cfg := postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      conf.PostgreSQL.DB,
		SSLMode: conf.PostgreSQL.SSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return fmt.Errorf("new postgres client: %w", err)
	}

	db := db.Database{Orm: orm}
	fmt.Println("Connected to the postgres database: ", conf.PostgreSQL.DB)

	err = db.Initialize()
	if err != nil {
		return fmt.Errorf("new postgres client: %w", err)
	}

	errors := make(chan error, 1)
	go func() {
		routes := httpRoutes{
			logger:             logger,
			db:                 db,
		}
		errors <- fmt.Errorf("http server: %w", httpserver.RegisterAndStart(ctx, logger, httpServerAddress, &routes))
	}()

	return <-errors


	
}


