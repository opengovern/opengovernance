package entity

import "github.com/kaytu-io/kaytu-util/pkg/es"

type IngestRequest struct {
	Doc es.Doc `json:"doc"`
}
