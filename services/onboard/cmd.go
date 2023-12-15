package onboard

import (
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/onboard/api"
	config2 "github.com/kaytu-io/kaytu-engine/services/onboard/config"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	var cnf config2.OnboardConfig
	config.ReadFromEnv(&cnf, nil)

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			// setup source events queue
			var qCfg queue.Config
			qCfg.Server.Username = cnf.RabbitMQ.Username
			qCfg.Server.Password = cnf.RabbitMQ.Password
			qCfg.Server.Host = cnf.RabbitMQ.Service
			qCfg.Server.Port = 5672
			qCfg.Queue.Name = config2.SourceEventsQueueName
			qCfg.Queue.Durable = true
			qCfg.Producer.ID = "ngonboard-service"
			q, err := queue.New(qCfg)
			if err != nil {
				return err
			}

			api.New(logger, q)

			cmd.SilenceUsage = true

			return httpserver.RegisterAndStart(logger, cnf.Http.Address, nil)
		},
	}

	return cmd
}
