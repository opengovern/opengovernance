package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/opengovern/og-util/pkg/api"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Database struct {
	Orm *gorm.DB
}

func (db Database) Initialize() error {
	err := db.Orm.AutoMigrate(
		&ApiKey{},
		&User{},
		&Configuration{},
		&Connector{},
	)
	if err != nil {
		return err
	}

	return nil
}
