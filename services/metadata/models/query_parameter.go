package models

import "github.com/opengovern/opengovernance/services/metadata/api"

type QueryParameter struct {
	Key   string `gorm:"primaryKey"`
	Value string `gorm:"type:text;not null"`
}

func (qp QueryParameter) GetKey() string {
	return qp.Key
}

func (qp QueryParameter) GetValue() string {
	return qp.Value
}

func (qp QueryParameter) ToAPI() api.QueryParameter {
	return api.QueryParameter{
		Key:   qp.Key,
		Value: qp.Value,
	}
}

func QueryParameterFromAPI(apiQP api.QueryParameter) QueryParameter {
	var qp QueryParameter
	qp.Key = apiQP.Key
	qp.Value = apiQP.Value
	return qp
}
