package assistant

import (
	"context"
	"github.com/kaytu-io/kaytu-engine/services/assistant/db"
	"github.com/kaytu-io/kaytu-engine/services/assistant/model"
	"github.com/kaytu-io/kaytu-engine/services/assistant/repository"
	"github.com/kaytu-io/kaytu-engine/services/migrator/config"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"go.uber.org/zap"
	"os"
	"path"
)

type Migration struct {
}

func (m Migration) IsGitBased() bool {
	return false
}

func (m Migration) AttachmentFolderPath() string {
	return "/workspace-migration"
}

func (m Migration) Run(conf config.MigratorConfig, logger *zap.Logger) error {
	database, err := db.New(koanf.Postgres{
		Host:     conf.PostgreSQL.Host,
		Port:     conf.PostgreSQL.Port,
		DB:       "assistant",
		Username: conf.PostgreSQL.Username,
		Password: conf.PostgreSQL.Password,
		SSLMode:  conf.PostgreSQL.SSLMode,
	}, logger)
	if err != nil {
		return err
	}
	promptRepo := repository.NewPrompt(database)

	prompt, err := os.ReadFile(path.Join(m.AttachmentFolderPath(), "chat_prompt.txt"))
	if err != nil {
		return err
	}

	err = promptRepo.Create(context.Background(), model.Prompt{
		Purpose: model.Purpose_ChatPrompt,
		Content: string(prompt),
	})
	if err != nil {
		return err
	}

	prompt, err = os.ReadFile(path.Join(m.AttachmentFolderPath(), "main_prompt.txt"))
	if err != nil {
		return err
	}

	err = promptRepo.Create(context.Background(), model.Prompt{
		Purpose: model.Purpose_SystemPrompt,
		Content: string(prompt),
	})
	if err != nil {
		return err
	}

	return nil
}
