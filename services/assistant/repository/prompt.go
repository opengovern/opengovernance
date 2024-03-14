package repository

import (
	"context"
	"errors"
	"github.com/kaytu-io/kaytu-engine/services/assistant/db"
	"github.com/kaytu-io/kaytu-engine/services/assistant/model"
	"gorm.io/gorm/clause"
)

var (
	ErrDuplicatePrompt = errors.New("didn't create run due to id conflict")
)

type Prompt interface {
	Create(context.Context, model.Prompt) error
	List(context.Context) ([]model.Prompt, error)
	UpdateContent(ctx context.Context, purpose model.Purpose, content string) error
}

type PromptSQL struct {
	db db.Database
}

func NewPrompt(db db.Database) Prompt {
	return PromptSQL{
		db: db,
	}
}

func (s PromptSQL) List(ctx context.Context) ([]model.Prompt, error) {
	var prompts []model.Prompt

	tx := s.db.DB.WithContext(ctx).Find(&prompts)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return prompts, nil
}

func (s PromptSQL) Create(ctx context.Context, c model.Prompt) error {
	tx := s.db.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "purpose"}},
			DoUpdates: clause.AssignmentColumns([]string{"content"}),
		}).
		Create(&c)

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return ErrDuplicatePrompt
	}

	return nil
}

func (s PromptSQL) UpdateContent(ctx context.Context, purpose model.Purpose, content string) error {
	tx := s.db.DB.WithContext(ctx).
		Model(&model.Prompt{}).
		Where("purpose = ?", purpose).
		Updates(model.Prompt{Purpose: purpose, Content: content})

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return ErrDuplicatePrompt
	}

	return nil
}
