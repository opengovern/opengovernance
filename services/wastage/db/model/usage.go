package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Usage struct {
	gorm.Model

	Request      datatypes.JSON
	Response     datatypes.JSON
	ResponseTime *float64 //Seconds
}
