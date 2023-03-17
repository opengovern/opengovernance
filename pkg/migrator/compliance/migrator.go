package compliance

import (
	"github.com/go-git/go-git/v5"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/db"
	"os"
)

func Run(db db.Database, complianceInputGitURL, queryInputGitURL string) error {
	compliancePath := "/tmp/loader-compliance-git"
	queryPath := "/tmp/loader-query-git"

	os.RemoveAll(compliancePath)
	_, err := git.PlainClone(compliancePath, false, &git.CloneOptions{
		URL:      complianceInputGitURL,
		Progress: os.Stdout,
	})
	if err != nil {
		return err
	}

	os.RemoveAll(queryPath)
	_, err = git.PlainClone(queryPath, false, &git.CloneOptions{
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
