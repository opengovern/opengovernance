package compliance

import (
	"github.com/go-git/go-git/v5"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/db"
	"os"
)

func Run(db db.Database, complianceInputGitURL string) error {
	os.RemoveAll("/tmp/loader-input-git")
	_, err := git.PlainClone("/tmp/loader-input-git", false, &git.CloneOptions{
		URL:      complianceInputGitURL,
		Progress: os.Stdout,
	})
	if err != nil {
		return err
	}

	err = PopulateDatabase(db.ORM)
	if err != nil {
		return err
	}

	return nil
}
