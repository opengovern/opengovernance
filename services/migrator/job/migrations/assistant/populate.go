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

func (m Migration) Run(ctx context.Context, conf config.MigratorConfig, logger *zap.Logger) error {
	database, err := db.New(koanf.Postgres{
		Host:     conf.PostgreSQL.Host,
		Port:     conf.PostgreSQL.Port,
		DB:       "assistant",
		Username: conf.PostgreSQL.Username,
		Password: conf.PostgreSQL.Password,
		SSLMode:  conf.PostgreSQL.SSLMode,
	}, logger)
	if err != nil {
		logger.Error("failed to create database", zap.Error(err))
		return err
	}
	promptRepo := repository.NewPrompt(database)

	prompt, err := os.ReadFile(path.Join(m.AttachmentFolderPath(), "chat_prompt.txt"))
	if err != nil {
		logger.Error("failed to read chat prompt", zap.Error(err))
		return err
	}

	err = promptRepo.Create(ctx, model.Prompt{
		Purpose:       model.Purpose_ChatPrompt,
		AssistantName: model.AssistantTypeQuery,
		Content:       string(prompt),
	})
	if err != nil {
		logger.Error("failed to create chat prompt", zap.Error(err))
		return err
	}

	prompt, err = os.ReadFile(path.Join(m.AttachmentFolderPath(), "main_prompt.txt"))
	if err != nil {
		logger.Error("failed to read main prompt", zap.Error(err))
		return err
	}

	err = promptRepo.Create(ctx, model.Prompt{
		Purpose:       model.Purpose_SystemPrompt,
		AssistantName: model.AssistantTypeQuery,
		Content:       string(prompt),
	})
	if err != nil {
		logger.Error("failed to create main prompt", zap.Error(err))
		return err
	}

	prompt, err = os.ReadFile(path.Join(m.AttachmentFolderPath(), "assets_chat_prompt.txt"))
	if err != nil {
		logger.Error("failed to read assets chat prompt", zap.Error(err))
		return err
	}

	err = promptRepo.Create(ctx, model.Prompt{
		Purpose:       model.Purpose_ChatPrompt,
		AssistantName: model.AssistantTypeAssets,
		Content:       string(prompt),
	})
	if err != nil {
		logger.Error("failed to create assets chat prompt", zap.Error(err))
		return err
	}

	prompt, err = os.ReadFile(path.Join(m.AttachmentFolderPath(), "assets_main_prompt.txt"))
	if err != nil {
		logger.Error("failed to read assets main prompt", zap.Error(err))
		return err
	}

	err = promptRepo.Create(ctx, model.Prompt{
		Purpose:       model.Purpose_SystemPrompt,
		AssistantName: model.AssistantTypeAssets,
		Content:       string(prompt),
	})
	if err != nil {
		logger.Error("failed to create assets main prompt", zap.Error(err))
		return err
	}

	// Score assistant
	prompt, err = os.ReadFile(path.Join(m.AttachmentFolderPath(), "score_chat_prompt.txt"))
	if err != nil {
		logger.Error("failed to read score chat prompt", zap.Error(err))
		return err
	}

	err = promptRepo.Create(ctx, model.Prompt{
		Purpose:       model.Purpose_ChatPrompt,
		AssistantName: model.AssistantTypeScore,
		Content:       string(prompt),
	})
	if err != nil {
		logger.Error("failed to create score chat prompt", zap.Error(err))
		return err
	}

	prompt, err = os.ReadFile(path.Join(m.AttachmentFolderPath(), "score_main_prompt.txt"))
	if err != nil {
		logger.Error("failed to read score main prompt", zap.Error(err))
		return err
	}

	err = promptRepo.Create(ctx, model.Prompt{
		Purpose:       model.Purpose_SystemPrompt,
		AssistantName: model.AssistantTypeScore,
		Content:       string(prompt),
	})
	if err != nil {
		logger.Error("failed to create score main prompt", zap.Error(err))
		return err
	}

	// Compliance assistant
	prompt, err = os.ReadFile(path.Join(m.AttachmentFolderPath(), "compliance_chat_prompt.txt"))
	if err != nil {
		logger.Error("failed to read compliance chat prompt", zap.Error(err))
		return err
	}

	err = promptRepo.Create(ctx, model.Prompt{
		Purpose:       model.Purpose_ChatPrompt,
		AssistantName: model.AssistantTypeCompliance,
		Content:       string(prompt),
	})
	if err != nil {
		logger.Error("failed to create compliance chat prompt", zap.Error(err))
		return err
	}

	prompt, err = os.ReadFile(path.Join(m.AttachmentFolderPath(), "compliance_main_prompt.txt"))
	if err != nil {
		logger.Error("failed to read compliance main prompt", zap.Error(err))
		return err
	}

	err = promptRepo.Create(ctx, model.Prompt{
		Purpose:       model.Purpose_SystemPrompt,
		AssistantName: model.AssistantTypeCompliance,
		Content:       string(prompt),
	})
	if err != nil {
		logger.Error("failed to create compliance main prompt", zap.Error(err))
		return err
	}

	return nil
}
