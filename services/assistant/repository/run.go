package repository

import (
	"context"
	"errors"
	"github.com/sashabaranov/go-openai"

	"github.com/kaytu-io/kaytu-engine/services/assistant/db"
	"github.com/kaytu-io/kaytu-engine/services/assistant/model"
	"gorm.io/gorm/clause"
)

var (
	ErrDuplicateThread = errors.New("didn't create run due to id conflict")
)

type Run interface {
	Get(context.Context, []string) ([]model.Run, error)
	Create(context.Context, model.Run) error
	List(context.Context) ([]model.Run, error)
	UpdateStatus(ctx context.Context, id string, threadID string, status openai.RunStatus) error
}

type RunSQL struct {
	db db.Database
}

func NewRun(db db.Database) Run {
	return RunSQL{
		db: db,
	}
}

func (s RunSQL) Get(ctx context.Context, ids []string) ([]model.Run, error) {
	var thread []model.Run

	tx := s.db.DB.WithContext(ctx).Find(&thread, "id IN ?", ids)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return thread, nil
}

func (s RunSQL) List(ctx context.Context) ([]model.Run, error) {
	var thread []model.Run

	tx := s.db.DB.WithContext(ctx).Find(&thread)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return thread, nil
}

func (s RunSQL) Create(ctx context.Context, c model.Run) error {
	tx := s.db.DB.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&c)

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return ErrDuplicateThread
	}

	return nil
}

func (s RunSQL) UpdateStatus(ctx context.Context, id string, threadID string, status openai.RunStatus) error {
	tx := s.db.DB.WithContext(ctx).
		Model(&model.Run{}).
		Where("id = ?", id).
		Where("thread_id = ?", threadID).
		Update("status", status)

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return ErrDuplicateThread
	}

	return nil
}
