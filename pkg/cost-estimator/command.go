package cost_estimator

import (
	"context"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"strconv"
)

var (
	KafkaService = os.Getenv("KAFKA_SERVICE")
	KafkaTopic   = os.Getenv("KAFKA_TOPIC")

	ElasticSearchAddress         = os.Getenv("ES_ADDRESS")
	ElasticSearchUsername        = os.Getenv("ES_USERNAME")
	ElasticSearchPassword        = os.Getenv("ES_PASSWORD")
	ElasticSearchIsOpenSearchStr = os.Getenv("ES_ISOPENSEARCH")
	ElasticSearchAwsRegion       = os.Getenv("ES_AWS_REGION")

	WorkspaceClientURL = os.Getenv("WORKSPACE_BASE_URL")

	HttpAddress = os.Getenv("HTTP_ADDRESS")
)

func Command() *cobra.Command {
	return &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			return start(cmd.Context())
		},
	}
}

func start(ctx context.Context) error {
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}

	elasticSearchIsOpenSearch, _ := strconv.ParseBool(ElasticSearchIsOpenSearchStr)
	esConf := config.ElasticSearch{
		Address:      ElasticSearchAddress,
		Username:     ElasticSearchUsername,
		Password:     ElasticSearchPassword,
		IsOpenSearch: elasticSearchIsOpenSearch,
		AwsRegion:    ElasticSearchAwsRegion,
	}
	handler, err := InitializeHttpHandler(
		WorkspaceClientURL, esConf,
		logger,
	)
	if err != nil {
		return fmt.Errorf("init http handler: %w", err)
	}

	return httpserver.RegisterAndStart(logger, HttpAddress, handler)
}
