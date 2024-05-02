package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UsageV2 struct {
	gorm.Model

	RequestId      *string
	ResponseId     *string
	ApiEndpoint    string
	Request        datatypes.JSON
	Response       datatypes.JSON
	FailureMessage *string
	Latency        *float64 //Seconds
	CliVersion     *string
}
