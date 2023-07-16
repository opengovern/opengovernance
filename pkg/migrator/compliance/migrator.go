package compliance

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/db"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/internal"

	"os"
)

func Run(db db.Database, complianceInputGitURLs []string, queryInputGitURL, githubToken string) error {
	os.RemoveAll(internal.ComplianceGitPath)
	for _, complianceInputGitURL := range complianceInputGitURLs {
		_, err := git.PlainClone(internal.ComplianceGitPath, false, &git.CloneOptions{
			Auth: &http.BasicAuth{
				Username: "abc123",
				Password: githubToken,
			},
			URL:      complianceInputGitURL,
			Progress: os.Stdout,
		})
		if err != nil {
			return err
		}
	}

	os.RemoveAll(internal.QueriesGitPath)
	_, err := git.PlainClone(internal.QueriesGitPath, false, &git.CloneOptions{
		Auth: &http.BasicAuth{
			Username: "abc123",
			Password: githubToken,
		},
		URL:      queryInputGitURL,
		Progress: os.Stdout,
	})
	if err != nil {
		return err
	}

	err = PopulateDatabase(db.ORM, internal.ComplianceGitPath, internal.QueriesGitPath)
	if err != nil {
		return err
	}

	return nil
}
