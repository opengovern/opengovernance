package job

import (
	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/client"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	"github.com/kaytu-io/kaytu-engine/services/migrator/config"
	"go.uber.org/zap"
	"os"
)

func GitClone(conf config.MigratorConfig, logger *zap.Logger) (string, error) {
	gitConfig := GitConfig{
		AnalyticsGitURL: conf.AnalyticsGitURL,
		githubToken:     conf.GithubToken,
	}

	metadataClient := client.NewMetadataServiceClient(conf.Metadata.BaseURL)
	value, err := metadataClient.GetConfigMetadata(&httpclient.Context{
		UserRole: api.AdminRole,
	}, models.MetadataKeyAnalyticsGitURL)

	if err == nil && len(value.GetValue().(string)) > 0 {
		gitConfig.AnalyticsGitURL = value.GetValue().(string)
	} else if err != nil {
		logger.Error("failed to get analytics git url from metadata", zap.Error(err))
	}

	logger.Info("using git repo", zap.String("url", gitConfig.AnalyticsGitURL))

	os.RemoveAll(config.AnalyticsGitPath)
	res, err := git.PlainClone(config.AnalyticsGitPath, false, &git.CloneOptions{
		Auth: &githttp.BasicAuth{
			Username: "abc123",
			Password: gitConfig.githubToken,
		},
		URL:      gitConfig.AnalyticsGitURL,
		Progress: os.Stdout,
	})
	if err != nil {
		logger.Error("Failure while running analytics migration", zap.Error(err))
		return "", err
	}

	ref, err := res.Head()
	if err != nil {
		logger.Error("failed to get head", zap.Error(err))
		return "", err
	}

	return ref.Hash().String(), nil
}
