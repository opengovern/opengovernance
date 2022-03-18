package inventory

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SmartQuery struct {
	gorm.Model
	ID          uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Provider    string
	Title       string
	Description string
	Query       string
}
