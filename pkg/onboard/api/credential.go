package api

import "gitlab.com/keibiengine/keibi-engine/pkg/source"

type CreateCredentialRequest struct {
	Name       string      `json:"name"`
	SourceType source.Type `json:"source_type"`
	Config     any         `json:"config"`
}

type CreateCredentialResponse struct {
	ID string `json:"id"`
}
