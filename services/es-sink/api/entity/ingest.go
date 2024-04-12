package entity

import "github.com/kaytu-io/kaytu-util/pkg/es"

type IngestRequest struct {
	Docs []es.Doc `json:"doc"`
}
