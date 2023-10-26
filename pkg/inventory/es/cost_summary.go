package es

import (
	"context"
	summarizer "github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
)

type ConnectionCostSummaryQueryResponse struct {
	Hits  ConnectionCostSummaryQueryHits `json:"hits"`
	PitID string                         `json:"pit_id"`
}

type ConnectionCostSummaryQueryHits struct {
	Total kaytu.SearchTotal               `json:"total"`
	Hits  []ConnectionCostSummaryQueryHit `json:"hits"`
}

type ConnectionCostSummaryQueryHit struct {
	ID      string                        `json:"_id"`
	Score   float64                       `json:"_score"`
	Index   string                        `json:"_index"`
	Type    string                        `json:"_type"`
	Version int64                         `json:"_version,omitempty"`
	Source  summarizer.ServiceCostSummary `json:"_source"`
	Sort    []any                         `json:"sort"`
}

type ConnectionCostPaginator struct {
	paginator *kaytu.BaseESPaginator
}

func NewConnectionCostPaginator(client kaytu.Client, filters []kaytu.BoolFilter, limit *int64) (ConnectionCostPaginator, error) {
	paginator, err := kaytu.NewPaginator(client.ES(), summarizer.CostSummeryIndex, filters, limit)
	if err != nil {
		return ConnectionCostPaginator{}, err
	}

	p := ConnectionCostPaginator{
		paginator: paginator,
	}

	return p, nil
}

func (p ConnectionCostPaginator) HasNext() bool {
	return !p.paginator.Done()
}

func (p ConnectionCostPaginator) NextPage(ctx context.Context) ([]summarizer.ServiceCostSummary, error) {
	var response ConnectionCostSummaryQueryResponse
	err := p.paginator.Search(ctx, &response)
	if err != nil {
		return nil, err
	}

	var values []summarizer.ServiceCostSummary
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
