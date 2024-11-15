package query_validator

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/es"
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	integration_type "github.com/opengovern/opengovernance/services/integration/integration-type"
	"github.com/opengovern/opengovernance/services/inventory/api"
	"go.uber.org/zap"
	"net/http"
	"regexp"
	"strings"
)

type QueryType string

const (
	QueryTypeNamedQuery        QueryType = "NAMED_QUERY"
	QueryTypeComplianceControl QueryType = "COMPLIANCE_CONTROL"
)

type Job struct {
	ID uint `json:"id"`

	QueryType       QueryType            `json:"query_type"`
	ControlId       string               `json:"control_id"`
	QueryId         string               `json:"query_id"`
	Parameters      []api.QueryParameter `json:"parameters"`
	Query           string               `json:"query"`
	IntegrationType []integration.Type   `json:"integration_type"`
	PrimaryTable    *string              `json:"primary_table"`
	ListOfTables    []string             `json:"list_of_tables"`
}

func (w *Worker) RunJob(ctx context.Context, job Job) error {
	ctx, cancel := context.WithTimeout(ctx, JobTimeout)
	defer cancel()
	res, err := w.RunSQLNamedQuery(ctx, job.Query)
	if err != nil {
		return err
	}

	if job.QueryType == QueryTypeComplianceControl {
		queryResourceType := ""
		if job.PrimaryTable != nil || len(job.ListOfTables) == 1 {
			tableName := ""
			if job.PrimaryTable != nil {
				tableName = *job.PrimaryTable
			} else {
				tableName = job.ListOfTables[0]
			}
			if tableName != "" {
				queryResourceType, _, err = GetResourceTypeFromTableName(tableName, job.IntegrationType)
				if err != nil {
					return err
				}
			}
		}
		if queryResourceType == "" {
			return fmt.Errorf(string(MissingResourceTypeQueryError))
		}

		esIndex := ResourceTypeToESIndex(queryResourceType)

		for _, record := range res.Data {
			if len(record) != len(res.Headers) {
				return fmt.Errorf("invalid record length, record=%d headers=%d", len(record), len(res.Headers))
			}
			recordValue := make(map[string]any)
			for idx, header := range res.Headers {
				value := record[idx]
				recordValue[header] = value
			}

			var platformResourceID string
			if v, ok := recordValue["og_resource_id"].(string); ok {
				platformResourceID = v
			} else {
				return fmt.Errorf(string(MissingPlatformResourceIDQueryError))
			}
			if _, ok := recordValue["og_account_id"].(string); !ok {
				return fmt.Errorf(string(MissingAccountIDQueryError))
			}
			if v, ok := recordValue["resource"].(string); !ok || v == "" || v == "null" {
				return fmt.Errorf(string(MissingResourceQueryError))
			}
			err = w.SearchResourceTypeByPlatformID(ctx, esIndex, platformResourceID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GetResourceTypeFromTableName(tableName string, queryIntegrationType []integration.Type) (string, integration.Type, error) {
	var integrationType integration.Type
	if len(queryIntegrationType) == 1 {
		integrationType = queryIntegrationType[0]
	} else {
		integrationType = ""
	}
	integration, ok := integration_type.IntegrationTypes[integrationType]
	if !ok {
		return "", "", echo.NewHTTPError(http.StatusInternalServerError, "unknown integration type")
	}
	return integration.GetResourceTypeFromTableName(tableName), integrationType, nil
}

var stopWordsRe = regexp.MustCompile(`\W+`)

func ResourceTypeToESIndex(t string) string {
	t = stopWordsRe.ReplaceAllString(t, "_")
	return strings.ToLower(t)
}

func (w *Worker) SearchResourceTypeByPlatformID(ctx context.Context, index string, platformID string) error {
	var filters []opengovernance.BoolFilter

	filters = append(filters, opengovernance.NewTermsFilter("platformResourceID", []string{platformID}))

	root := map[string]any{}

	root["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		w.logger.Error("SearchResourceTypeByPlatformID", zap.Error(err), zap.String("query", string(queryBytes)), zap.String("index", index))
		return err
	}

	w.logger.Info("SearchResourceTypeByPlatformID", zap.String("query", string(queryBytes)), zap.String("index", index))

	var resp SearchResourceTypeByPlatformIDResponse
	err = w.esClient.Search(ctx, index, string(queryBytes), &resp)
	if err != nil {
		w.logger.Error("SearchResourceTypeByPlatformID", zap.Error(err), zap.String("query", string(queryBytes)), zap.String("index", index))
		return err
	}
	if len(resp.Hits.Hits) > 0 {
		w.logger.Info("SearchResourceTypeByPlatformID", zap.String("query", string(queryBytes)), zap.String("index", index),
			zap.String("platformID", platformID), zap.Any("result", resp.Hits.Hits))
	} else {
		return fmt.Errorf(string(ResourceNotFoundQueryError))
	}
	return nil
}

type SearchResourceTypeByPlatformIDHit struct {
	ID      string      `json:"_id"`
	Score   float64     `json:"_score"`
	Index   string      `json:"_index"`
	Type    string      `json:"_type"`
	Version int64       `json:"_version,omitempty"`
	Source  es.Resource `json:"_source"`
	Sort    []any       `json:"sort"`
}

type SearchResourceTypeByPlatformIDResponse struct {
	Hits struct {
		Hits []SearchResourceTypeByPlatformIDHit `json:"hits"`
	} `json:"hits"`
}
