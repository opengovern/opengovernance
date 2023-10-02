package insight

import (
	"errors"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
)

const (
	InsightJobsQueueName    = "insight-jobs-queue"
	InsightResultsQueueName = "insight-results-queue"
)

var (
	SteampipeHost = os.Getenv("STEAMPIPE_HOST")

	S3Endpoint     = os.Getenv("S3_ENDPOINT")
	S3AccessKey    = os.Getenv("S3_ACCESS_KEY")
	S3AccessSecret = os.Getenv("S3_ACCESS_SECRET")
	S3Region       = os.Getenv("S3_REGION")
	S3Bucket       = os.Getenv("S3_BUCKET")

	CurrentWorkspaceID = os.Getenv("CURRENT_NAMESPACE")
)

func WorkerCommand() *cobra.Command {
	var (
		id  string
		cnf WorkerConfig
	)
	config.ReadFromEnv(&cnf, nil)

	cmd := &cobra.Command{
		PreRunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case id == "":
				return errors.New("missing required flag 'id'")
			default:
				return nil
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true

			w, err := InitializeWorker(
				id,
				cnf,
				InsightJobsQueueName,
				InsightResultsQueueName,
				logger,
				S3Endpoint, S3AccessKey,
				S3AccessSecret, S3Region,
				S3Bucket,
			)
			if err != nil {
				return err
			}

			defer w.Stop()

			return w.Run()
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "The worker id")

	return cmd
}
