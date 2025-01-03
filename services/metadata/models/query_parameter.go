package models

import "github.com/opengovern/opencomply/services/metadata/api"

type QueryParameterValues struct {
	Key       string `gorm:"primaryKey"`
	ControlID string `gorm:"primaryKey"`
	Value     string `gorm:"type:text;not null"`
}

func (qp QueryParameterValues) GetKey() string {
	return qp.Key
}

func (qp QueryParameterValues) GetValue() string {
	return qp.Value
}

func (qp QueryParameterValues) ToAPI() api.QueryParameter {
	return api.QueryParameter{
		Key:   qp.Key,
		Value: qp.Value,
	}
}

func QueryParameterFromAPI(apiQP api.QueryParameter) QueryParameterValues {
	var qp QueryParameterValues
	qp.Key = apiQP.Key
	qp.Value = apiQP.Value
	return qp
}
