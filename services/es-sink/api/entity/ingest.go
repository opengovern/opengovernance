package entity

import "github.com/kaytu-io/kaytu-util/pkg/es"

type IngestRequest struct {
	Docs []es.DocBase `json:"doc"`
}

type FailedDoc struct {
	Doc es.DocBase `json:"doc"`
	Err string     `json:"err"`
}

type IngestResponse struct {
	FailedDocs []FailedDoc `json:"failed_docs"`
}
