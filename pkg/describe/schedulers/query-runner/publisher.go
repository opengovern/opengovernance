package query_runner

import (
	"bytes"
	"context"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	query_runner "github.com/kaytu-io/open-governance/pkg/inventory/query-runner"
	"github.com/kaytu-io/open-governance/pkg/metadata/models"
	"go.uber.org/zap"
	"text/template"
)

func (s *JobScheduler) runPublisher(ctx context.Context) error {
	ctx2 := &httpclient.Context{UserRole: api.InternalRole}
	ctx2.Ctx = ctx

	s.logger.Info("Query Runner publisher started")
	jobs, err := s.db.FetchCreatedQueryRunnerJobs()
	if err != nil {
		s.logger.Error("Fetch Created Query Runner Jobs Error", zap.Error(err))
		return err
	}
	for _, job := range jobs {
		namedQuery, err := s.inventoryClient.GetQuery(ctx2, job.QueryId)
		if err != nil {
			s.logger.Error("Get Query Error", zap.Error(err))
		}
		controlQuery, err := s.complianceClient.GetControlDetails(ctx2, job.QueryId)
		if err != nil {
			s.logger.Error("Get Control Error", zap.Error(err))
		}
		var query string
		var parameters []models.QueryParameter
		if namedQuery != nil {
			query = namedQuery.Query.QueryToExecute
			parameters = namedQuery.Query.Parameters
		} else if controlQuery != nil {
			query = controlQuery.Query.QueryToExecute
		}
		s.logger.Info("Query Runner publisher", zap.String("query", query))

		queryParams, err := s.metadataClient.ListQueryParameters(&httpclient.Context{UserRole: api.InternalRole})
		if err != nil {
			return err
		}
		queryParamMap := make(map[string]string)
		for _, qp := range queryParams.QueryParameters {
			queryParamMap[qp.Key] = qp.Value
		}
		queryTemplate, err := template.New("query").Parse(query)
		if err != nil {
			return err
		}
		var queryOutput bytes.Buffer
		if err := queryTemplate.Execute(&queryOutput, queryParamMap); err != nil {
			return fmt.Errorf("failed to execute query template: %w", err)
		}

		runnerJobMsg := query_runner.Job{
			RunId:       job.RunId,
			RetryCount:  0,
			CreatedBy:   job.CreatedBy,
			TriggeredAt: job.CreatedAt.UnixMilli(),
			QueryId:     job.QueryId,
			Query:       query,
		}
	}
}
