package es

import (
	"context"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

type ConnectionResourceTypeQueryResponse struct {
	Hits  ConnectionResourceTypeQueryHits `json:"hits"`
	PitID string                          `json:"pit_id"`
}
type ConnectionResourceTypeQueryHits struct {
	Total keibi.SearchTotal                `json:"total"`
	Hits  []ConnectionResourceTypeQueryHit `json:"hits"`
}
type ConnectionResourceTypeQueryHit struct {
	ID      string                                `json:"_id"`
	Score   float64                               `json:"_score"`
	Index   string                                `json:"_index"`
	Type    string                                `json:"_type"`
	Version int64                                 `json:"_version,omitempty"`
	Source  es.ConnectionResourceTypeTrendSummary `json:"_source"`
	Sort    []any                                 `json:"sort"`
}

type ConnectionResourceTypePaginator struct {
	paginator *keibi.BaseESPaginator
}

func NewConnectionResourceTypePaginator(client keibi.Client, filters []keibi.BoolFilter, limit *int64) (ConnectionResourceTypePaginator, error) {
	paginator, err := keibi.NewPaginator(client.ES(), es.ConnectionSummaryIndex, filters, limit)
	if err != nil {
		return ConnectionResourceTypePaginator{}, err
	}

	p := ConnectionResourceTypePaginator{
		paginator: paginator,
	}

	return p, nil
}

func (p ConnectionResourceTypePaginator) HasNext() bool {
	return !p.paginator.Done()
}

func (p ConnectionResourceTypePaginator) NextPage(ctx context.Context) ([]es.ConnectionResourceTypeTrendSummary, error) {
	var response ConnectionResourceTypeQueryResponse
	err := p.paginator.Search(ctx, &response)
	if err != nil {
		return nil, err
	}

	var values []es.ConnectionResourceTypeTrendSummary
	for _, hit := range response.Hits.Hits {
		values = append(values, hit.Source)
	}

	hits := int64(len(response.Hits.Hits))
	if hits > 0 {
		p.paginator.UpdateState(hits, response.Hits.Hits[hits-1].Sort, response.PitID)
	} else {
		p.paginator.UpdateState(hits, nil, "")
	}

	return values, nil
}
