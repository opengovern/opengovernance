package job

import (
	"encoding/json"
	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/client"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	"github.com/kaytu-io/kaytu-engine/services/migrator/config"
	"github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"go.uber.org/zap"
	"os"
)

func GitClone(conf config.MigratorConfig, logger *zap.Logger) (string, error) {
	gitConfig := GitConfig{
		AnalyticsGitURL:         conf.AnalyticsGitURL,
		ControlEnrichmentGitURL: conf.ControlEnrichmentGitURL,
		githubToken:             conf.GithubToken,
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
	gitConfig.AnalyticsGitURL = "https://github.com/kaytu-io/configz-deprecated"

	logger.Info("using git repo", zap.String("url", gitConfig.AnalyticsGitURL))

	refs := make([]string, 0, 2)

	gitAuth := githttp.BasicAuth{
		Username: "abc123",
		Password: gitConfig.githubToken,
	}
	os.RemoveAll(config.ConfigzGitPath)
	res, err := git.PlainClone(config.ConfigzGitPath, false, &git.CloneOptions{
		Auth:     &gitAuth,
		URL:      gitConfig.AnalyticsGitURL,
		Progress: os.Stdout,
	})
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

	logger.Info("using git repo for enrichmentor", zap.String("url", gitConfig.ControlEnrichmentGitURL))

	os.RemoveAll(config.ControlEnrichmentGitPath)
	res, err = git.PlainClone(config.ControlEnrichmentGitPath, false, &git.CloneOptions{
		Auth:     &gitAuth,
		URL:      gitConfig.ControlEnrichmentGitURL,
		Progress: os.Stdout,
	})
	if err != nil {
		logger.Error("Failure while cloning control enrichment repo", zap.Error(err))
		return "", err
	}
	ref, err = res.Head()
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
