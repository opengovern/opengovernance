package insight

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/db"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/internal"

	"os"
)

func Run(db db.Database, insightsInputGitURL, githubToken string) error {
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
		return err
	}

	err = PopulateDatabase(db.ORM, internal.InsightsGitPath)
	if err != nil {
		return err
	}

	return nil
}
