package repository

import (
	"github.com/kaytu-io/kaytu-engine/services/integration/db"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Source struct {
	db db.Database
}

func NewSource(db db.Database) Source {
	return Source{
		db: db,
	}
}

// ListSources gets list of all source
func (s Source) List() ([]model.Source, error) {
	var sources []model.Source

	tx := s.db.DB.Model(model.Source{}).Joins("Connector").Joins("Credential").Find(&sources)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return sources, nil
}

// GetSourcesOfType gets list of sources with matching type
func (s Source) ListOfType(t source.Type) ([]model.Source, error) {
	return s.ListOfTypes([]source.Type{t})
}

// GetSourcesOfTypes gets list of sources with matching types
func (s Source) ListOfTypes(types []source.Type) ([]model.Source, error) {
	var sources []model.Source
	tx := s.db.DB.Joins("Connector").Joins("Credential").Find(&sources, "type IN ?", types)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return sources, nil
}
