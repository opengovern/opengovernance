package analytics

import (
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/internal"
	"go.uber.org/zap"

	"os"
)

func Run(logger *zap.Logger, conf postgres.Config, analyticsInputGitURL, githubToken string) error {
	conf.DB = "inventory"
	orm, err := postgres.NewClient(&conf, logger)
	if err != nil {
		logger.Error("failed to create postgres client", zap.Error(err))
		return fmt.Errorf("new postgres client: %w", err)
	}

	os.RemoveAll(internal.AnalyticsGitPath)
	_, err = git.PlainClone(internal.AnalyticsGitPath, false, &git.CloneOptions{
		Auth: &http.BasicAuth{
			Username: "abc123",
			Password: githubToken,
		},
		URL:      analyticsInputGitURL,
		Progress: os.Stdout,
	})
	if err != nil {
		return fmt.Errorf("failure in clone: %v", err)
	}

	err = PopulateDatabase(logger, orm, internal.AnalyticsGitPath)
	if err != nil {
		return err
	}

	return nil
}
