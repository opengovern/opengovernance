package repository

import (
	"context"
	"errors"

	"github.com/kaytu-io/kaytu-engine/services/assistant/db"
	"github.com/kaytu-io/kaytu-engine/services/assistant/model"
	"gorm.io/gorm/clause"
)

var (
	ErrDuplicateThread = errors.New("didn't create thread due to id conflict")
)

type Thread interface {
	Get(context.Context, []string) ([]model.Thread, error)
	Create(context.Context, model.Thread) error
}

type ThreadSQL struct {
	db db.Database
}

func NewThreadSQL(db db.Database) Thread {
	return ThreadSQL{
		db: db,
	}
}

// Get connections that their ID exist in the IDs list.
func (s ThreadSQL) Get(ctx context.Context, ids []string) ([]model.Thread, error) {
	var thread []model.Thread

	tx := s.db.DB.WithContext(ctx).Find(&thread, "id IN ?", ids)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return thread, nil
}

func (s ThreadSQL) Create(ctx context.Context, c model.Thread) error {
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
