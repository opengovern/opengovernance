package types

import (
	"github.com/kaytu-io/kaytu-engine/services/migrator/config"
	"go.uber.org/zap"
)

type Migration interface {
	Run(conf config.MigratorConfig, logger *zap.Logger) error
	IsGitBased() bool
	AttachmentFolderPath() string
}
