package describer

import (
	"context"
	"errors"
	"fmt"
	hamiltonAuth "github.com/manicminer/hamilton/auth"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resourcegraph/mgmt/resourcegraph"
	"github.com/Azure/go-autorest/autorest"
)

const SubscriptionBatchSize = 100

type GenericResourceGraph struct {
	Table string
	Type  string
}

func (d GenericResourceGraph) DescribeResources(ctx context.Context, authorizer autorest.Authorizer, _ hamiltonAuth.Authorizer, subscriptions []string, tenantId string) ([]Resource, error) {
	query := fmt.Sprintf("%s | where type == \"%s\"", d.Table, strings.ToLower(d.Type))

	client := resourcegraph.New()
	client.Authorizer = authorizer

	values := []Resource{}

	// Group the subscriptions to batches with a max size
	for i := 0; i < len(subscriptions); i = i + SubscriptionBatchSize {
		j := i + SubscriptionBatchSize
		if j > len(subscriptions) {
			j = len(subscriptions)
		}

		subs := subscriptions[i:j]
		request := resourcegraph.QueryRequest{
			Subscriptions: &subs,
			Query:         &query,
			Options: &resourcegraph.QueryRequestOptions{
				ResultFormat: resourcegraph.ResultFormatObjectArray,
			},
		}

		// Fetch all resources by paging through all the results
		for first, skipToken := true, (*string)(nil); first || skipToken != nil; {
			request.Options.SkipToken = skipToken

			response, err := client.Resources(ctx, request)
			if err != nil {
				return nil, err
			}

			quotaRemaining, untilResets, err := quota(response.Header)
			if err != nil {
				return nil, err
			}

			if quotaRemaining == 0 {
				time.Sleep(untilResets)
			}

			for _, v := range response.Data.([]interface{}) {
				m := v.(map[string]interface{})
				values = append(values, Resource{
					ID:          m["id"].(string),
					Description: v,
				})
			}
			first, skipToken = false, response.SkipToken
		}
	}

	return values, nil
}

// quota parses the Azure throttling headers.
// See https://docs.microsoft.com/en-us/azure/governance/resource-graph/concepts/guidance-for-throttled-requests#understand-throttling-headers
func quota(header http.Header) (int, time.Duration, error) {
	remainingHeader := header[http.CanonicalHeaderKey("x-ms-user-quota-remaining")]
	if len(remainingHeader) == 0 {
		return 0, 0, errors.New("header 'x-ms-user-quota-remaining' missing")
	}

	remaining, err := strconv.Atoi(remainingHeader[0])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse 'x-ms-user-quota-remaining':  %w", err)
	}

	afterHeader := header[http.CanonicalHeaderKey("x-ms-user-quota-resets-after")]
	if len(afterHeader) == 0 {
		return 0, 0, errors.New("header 'x-ms-user-quota-resets-after' missing")
	}

	t, err := time.Parse("15:04:05", afterHeader[0])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse 'x-ms-user-quota-resets-after'")
	}

	t = t.UTC()
	after := time.Duration(t.Second())*time.Second +
		time.Duration(t.Minute())*time.Minute +
		time.Duration(t.Hour())*time.Hour

	return remaining, after, nil
}
