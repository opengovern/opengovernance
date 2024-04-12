package entity

import "github.com/kaytu-io/kaytu-util/pkg/es"

type IngestRequest struct {
	Docs []es.DocBase `json:"doc"`
}
