package keibi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"strings"

	elasticsearchv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"github.com/turbot/steampipe-plugin-sdk/plugin"
)

func closeSafe(resp *esapi.Response) {
	if resp != nil && resp.Body != nil {
		_, _ = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}
}

func checkError(resp *esapi.Response) error {
	if !resp.IsError() {
		return nil
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}

	var e ErrorResponse
	if err := json.Unmarshal(data, &e); err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}

	return e
}

func isIndexNotFoundErr(err error) bool {
	var e ErrorResponse
	return errors.As(err, &e) &&
		strings.EqualFold(e.Info.Type, "index_not_found_exception")
}

type BoolFilter interface {
	IsBoolFilter()
}

func buildFilter(equalQuals plugin.KeyColumnEqualsQualMap, filtersQuals map[string]string) []BoolFilter {
	var filters []BoolFilter
	for columnName, filterName := range filtersQuals {
		if equalQuals[columnName] == nil {
			continue
		}

		var filter BoolFilter
		value := equalQuals[columnName]
		if value.GetStringValue() != "" {
			filter = TermFilter(filterName, equalQuals[columnName].GetStringValue())
		} else if value.GetListValue() != nil {
			list := value.GetListValue()
			values := make([]string, 0, len(list.Values))
			for _, value := range list.Values {
				values = append(values, value.GetStringValue())
			}

			filter = TermsFilter(filterName, values)
		}

		filters = append(filters, filter)
	}
	return filters
}

type termFilter struct {
	field string
	value string
}

func TermFilter(field, value string) BoolFilter {
	return termFilter{
		field: field,
		value: value,
	}
}

func (t termFilter) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"term": map[string]string{
			t.field: t.value,
		},
	})
}

func (t termFilter) IsBoolFilter() {}

type termsFilter struct {
	field  string
	values []string
}

func TermsFilter(field string, values []string) BoolFilter {
	return termsFilter{
		field:  field,
		values: values,
	}
}

func (t termsFilter) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"terms": map[string][]string{
			t.field: t.values,
		},
	})
}

func (t termsFilter) IsBoolFilter() {}

type baseESPaginator struct {
	client *elasticsearchv7.Client

	index    string                 // Query index
	query    map[string]interface{} // Query filters
	pageSize int64                  // Query page size
	pitID    string                 // Query point in time id (Only set if max is greater than size)

	limit   int64 // Maximum documents to query
	queried int64 // Current count of queried documents

	searchAfter []interface{}
	done        bool
}

func newPaginator(client *elasticsearchv7.Client, index string, filters []BoolFilter, limit *int64) (*baseESPaginator, error) {
	var query map[string]interface{}
	if len(filters) > 0 {
		query = map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filters,
			},
		}
	} else {
		query = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}

	var max int64
	if limit == nil {
		max = math.MaxInt64
	} else {
		max = *limit
	}

	if max < 0 {
		return nil, fmt.Errorf("invalid limit: %d", max)
	}

	return &baseESPaginator{
		client:   client,
		index:    index,
		query:    query,
		pageSize: 10_000,
		limit:    max,
		queried:  0,
	}, nil
}

// The response will be marshalled if the search was successfull
func (p *baseESPaginator) search(ctx context.Context, response interface{}) error {
	if p.done {
		return errors.New("no more page to query")
	}

	if err := p.createPit(ctx); err != nil {
		if isIndexNotFoundErr(err) {
			return nil
		}
		return err
	}

	sa := SearchRequest{
		Size:  &p.pageSize,
		Query: p.query,
		Sort: []map[string]interface{}{
			{
				"_shard_doc": "desc",
			},
		},
	}

	if p.limit > p.pageSize {
		sa.PIT = &PointInTime{
			ID:        p.pitID,
			KeepAlive: "1m",
		}
	}

	if p.searchAfter != nil {
		sa.SearchAfter = p.searchAfter
	}

	opts := []func(*esapi.SearchRequest){
		p.client.Search.WithContext(ctx),
		p.client.Search.WithBody(esutil.NewJSONReader(sa)),
		p.client.Search.WithTrackTotalHits(false),
	}
	if sa.PIT == nil {
		opts = append(opts, p.client.Search.WithIndex(p.index))
	}

	res, err := p.client.Search(opts...)
	defer closeSafe(res)
	if err != nil {
		return err
	} else if err := checkError(res); err != nil {
		if isIndexNotFoundErr(err) {
			return nil
		}
		return err
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if err := json.Unmarshal(b, response); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	return nil
}

// createPit, sets up the PointInTime for the search with more than 10000 limit
func (p *baseESPaginator) createPit(ctx context.Context) error {
	if p.limit < p.pageSize {
		return nil
	} else if p.pitID != "" {
		return nil
	}

	resPit, err := p.client.OpenPointInTime([]string{p.index}, "1m",
		p.client.OpenPointInTime.WithContext(ctx),
	)
	defer closeSafe(resPit)
	if err != nil {
		return err
	} else if err := checkError(resPit); err != nil {
		return err
	}

	data, err := ioutil.ReadAll(resPit.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var pit PointInTimeResponse
	if err := json.Unmarshal(data, &pit); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	p.pitID = pit.ID
	return nil
}

func (p *baseESPaginator) updateState(numHits int64, searchAfter []interface{}, pitID string) {
	p.queried += numHits
	if p.queried > p.limit {
		// Have found enough documents
		p.done = true
	} else if numHits == 0 || numHits < p.pageSize {
		// The isn't more documents thus the last batch had less than page size
		p.done = true
	}

	if numHits > 0 {
		p.searchAfter = searchAfter
		p.pitID = pitID
	}
}
