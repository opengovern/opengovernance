package compliance

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/db"
	"os"
)

func Run(db db.Database, complianceInputGitURL, queryInputGitURL, githubToken string) error {
	compliancePath := "/tmp/loader-compliance-git"
	queryPath := "/tmp/loader-query-git"

	os.RemoveAll(compliancePath)
	_, err := git.PlainClone(compliancePath, false, &git.CloneOptions{
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

	os.RemoveAll(queryPath)
	_, err = git.PlainClone(queryPath, false, &git.CloneOptions{
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

	err = PopulateDatabase(db.ORM, compliancePath, queryPath)
	if err != nil {
		return err
	}

	return nil
}
