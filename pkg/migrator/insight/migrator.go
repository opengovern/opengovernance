package insight

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/db"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/internal"
	"go.uber.org/zap"

	"os"
)

func Run(logger *zap.Logger, db db.Database, insightsInputGitURL, githubToken string) error {
	os.RemoveAll(internal.InsightsGitPath)
	_, err := git.PlainClone(internal.InsightsGitPath, false, &git.CloneOptions{
		Auth: &http.BasicAuth{
			Username: "abc123",
			Password: githubToken,
		},
		URL:      insightsInputGitURL,
		Progress: os.Stdout,
	})
	if err != nil {
		return fmt.Errorf("failure in clone: %v", err)
	}

	err = PopulateDatabase(logger, db.ORM, internal.InsightsGitPath)
	if err != nil {
		return err
	}

	return nil
}
