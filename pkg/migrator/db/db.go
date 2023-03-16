package db

import (
	"gorm.io/gorm"
)

type Database struct {
	ORM *gorm.DB
}
