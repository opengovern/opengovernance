package repository

import (
	"github.com/kaytu-io/kaytu-engine/services/integration/db"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Connection interface {
	List() ([]model.Connection, error)
	ListOfType(source.Type) ([]model.Connection, error)
	ListOfTypes([]source.Type) ([]model.Connection, error)
}

type ConnectionSQL struct {
	db db.Database
}

func NewConnectionSQL(db db.Database) Connection {
	return ConnectionSQL{
		db: db,
	}
}

// ListSources gets list of all source
func (s ConnectionSQL) List() ([]model.Connection, error) {
	var connections []model.Connection

	tx := s.db.DB.Model(model.Connection{}).Joins("Connector").Joins("Credential").Find(&connections)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return connections, nil
}

// GetSourcesOfType gets list of sources with matching type
func (s ConnectionSQL) ListOfType(t source.Type) ([]model.Connection, error) {
	return s.ListOfTypes([]source.Type{t})
}

// GetSourcesOfTypes gets list of sources with matching types
func (s ConnectionSQL) ListOfTypes(types []source.Type) ([]model.Connection, error) {
	var connections []model.Connection
	tx := s.db.DB.Joins("Connector").Joins("Credential").Find(&connections, "type IN ?", types)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return connections, nil
}
