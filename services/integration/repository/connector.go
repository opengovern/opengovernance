package repository

import (
	"context"

	"github.com/opengovern/opengovernance/services/integration/db"
	"github.com/opengovern/opengovernance/services/integration/model"
)

type Connector interface {
	List(context.Context) ([]model.Connector, error)
}

type ConnectorSQL struct {
	db db.Database
}

func NewConnectorSQL(db db.Database) Connector {
	return ConnectorSQL{db: db}
}

func (s ConnectorSQL) List(ctx context.Context) ([]model.Connector, error) {
	var connectors []model.Connector

	if err := s.db.DB.WithContext(ctx).Find(&connectors).Error; err != nil {
		return nil, err
	}

	return connectors, nil
}
