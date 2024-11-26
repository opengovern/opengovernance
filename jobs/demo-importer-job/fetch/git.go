package fetch

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/opengovern/opencomply/jobs/demo-importer-job/types"
	"go.uber.org/zap"
)

type GitConfig struct {
	DemoDataGitURL string
	githubToken    string
}

func GitClone(conf types.DemoImporterConfig, logger *zap.Logger) (string, error) {
	gitConfig := GitConfig{
		DemoDataGitURL: conf.DemoDataGitURL,
		githubToken:    conf.GithubToken,
	}

	logger.Info("using git repo", zap.String("url", gitConfig.DemoDataGitURL))

	refs := make([]string, 0, 2)

	var authMethod transport.AuthMethod
	if gitConfig.githubToken != "" {
		authMethod = &githttp.BasicAuth{
			Username: "abc123",
			Password: gitConfig.githubToken,
		}
	}
	os.RemoveAll(types.DemoDataPath)
	co := git.CloneOptions{
		Auth:     authMethod,
		URL:      gitConfig.DemoDataGitURL,
		Progress: os.Stdout,
	}
	if strings.Contains(co.URL, "@") {
		newUrl, tag, _ := strings.Cut(co.URL, "@")
		co.URL = newUrl
		co.ReferenceName = plumbing.ReferenceName("refs/tags/" + tag)
	}
	res, err := git.PlainClone(types.DemoDataPath, false, &co)
	if err != nil {
		logger.Error("Failure while cloning analytics repo", zap.Error(err))
		return "", err
	}
	ref, err := res.Head()
	if err != nil {
		logger.Error("failed to get head", zap.Error(err))
		return "", err
	}
	refs = append(refs, ref.Hash().String())

	refsJson, err := json.Marshal(refs)
	if err != nil {
		logger.Error("failed to marshal refs", zap.Error(err))
		return "", err
	}

	return string(refsJson), nil
}
