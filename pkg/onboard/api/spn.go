package api

import (
	"encoding/json"

	"github.com/google/uuid"
)

type SPNConfigAzure struct {
	TenantId     string `json:"tenantId" validate:"required,uuid_rfc4122"`
	ClientId     string `json:"clientId" validate:"required"`
	ClientSecret string `json:"clientSecret" validate:"required"`
}

func (s SPNConfigAzure) AsMap() map[string]interface{} {
	in, err := json.Marshal(s)
	if err != nil {
		panic(err) // Don't expect any error
	}

	var out map[string]interface{}
	if err := json.Unmarshal(in, &out); err != nil {
		panic(err) // Don't expect any error
	}

	return out
}

type CreateSPNRequest struct {
	Config SPNConfigAzure `json:"config"`
}

type CreateSPNResponse struct {
	ID uuid.UUID `json:"id"`
}

type SPNCredential struct {
	SPNName      string `json:"spnName"`
	ClientID     string `json:"clientID"`
	TenantID     string `json:"tenantID"`
	ClientSecret string `json:"clientSecret"`
}

type SPNRecord struct {
	SPNID    string `json:"spnID"`
	SPNName  string `json:"spnName"`
	ClientID string `json:"clientID"`
	TenantID string `json:"tenantID"`
}
