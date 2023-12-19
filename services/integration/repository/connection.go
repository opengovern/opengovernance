package repository

import (
	"context"

	"github.com/kaytu-io/kaytu-engine/services/integration/db"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Connection interface {
	List(context.Context) ([]model.Connection, error)
	ListOfType(context.Context, source.Type) ([]model.Connection, error)
	ListOfTypes(context.Context, []source.Type) ([]model.Connection, error)
	Get(context.Context, []string) ([]model.Connection, error)
}

type ConnectionSQL struct {
	db db.Database
}

func NewConnectionSQL(db db.Database) Connection {
	return ConnectionSQL{
		db: db,
	}
}

// List gets list of all source
func (s ConnectionSQL) List(ctx context.Context) ([]model.Connection, error) {
	var connections []model.Connection

	tx := s.db.DB.WithContext(ctx).Joins("Connector").Joins("Credential").Find(&connections)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return connections, nil
}

// ListOfType gets list of sources with matching type
func (s ConnectionSQL) ListOfType(ctx context.Context, t source.Type) ([]model.Connection, error) {
	return s.ListOfTypes(ctx, []source.Type{t})
}

// ListOfTypes gets list of sources with matching types
func (s ConnectionSQL) ListOfTypes(ctx context.Context, types []source.Type) ([]model.Connection, error) {
	var connections []model.Connection

	tx := s.db.DB.WithContext(ctx).Joins("Connector").Joins("Credential").Find(&connections, "type IN ?", types)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return connections, nil
}

func (s ConnectionSQL) Get(ctx context.Context, ids []string) ([]model.Connection, error) {
	var connections []model.Connection

	tx := s.db.DB.WithContext(ctx).Joins("Connector").Joins("Credential").Find(&connections, "id IN ?", ids)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return connections, nil
}
