package compliancebuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	complianceApi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	complianceClient "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
)

type BenchmarkSummaryBuilder struct {
	jobID  uint
	logger *zap.Logger
	client keibi.Client

	policyMap map[string]complianceApi.Policy

	policySummaries    map[string]map[string]types.PolicySummary
	benchmarkSummaries map[string]map[string]types.BenchmarkSummary
	complianceClient   complianceClient.ComplianceServiceClient
}

func NewBenchmarkSummaryBuilder(logger *zap.Logger, jobId uint, client keibi.Client, complianceClient complianceClient.ComplianceServiceClient) *BenchmarkSummaryBuilder {
	return &BenchmarkSummaryBuilder{
		jobID:              jobId,
		logger:             logger,
		client:             client,
		complianceClient:   complianceClient,
		policyMap:          make(map[string]complianceApi.Policy),
		policySummaries:    make(map[string]map[string]types.PolicySummary),
		benchmarkSummaries: make(map[string]map[string]types.BenchmarkSummary),
	}
}

func (b *BenchmarkSummaryBuilder) Process(resource types.Finding) {
	if _, ok := b.policySummaries[resource.PolicyID]; !ok {
		b.policySummaries[resource.PolicyID] = make(map[string]types.PolicySummary)
	}
	if _, ok := b.policyMap[resource.PolicyID]; !ok {
		policy, err := b.complianceClient.GetPolicy(&httpclient.Context{UserRole: authApi.KeibiAdminRole}, resource.PolicyID)
		if err != nil {
			b.logger.Error("failed to get policy", zap.Error(err))
			return
		}
		if policy == nil {
			b.policyMap[resource.PolicyID] = *policy
		}
	}
	policy := b.policyMap[resource.PolicyID]
	if _, ok := b.policySummaries[resource.PolicyID][resource.ConnectionID]; !ok {
		b.policySummaries[resource.PolicyID][resource.ConnectionID] = types.PolicySummary{
			PolicyID:      resource.PolicyID,
			ConnectorType: resource.Connector,
			TotalResult:   types.ComplianceResultSummary{},
		}
	}
	policySummary := b.policySummaries[resource.PolicyID][resource.ConnectionID]
	switch resource.Result {
	case types.ComplianceResultOK:
		policySummary.TotalResult.OkCount++
		policySummary.TotalSeverity.PassedCount++
	case types.ComplianceResultALARM:
		policySummary.TotalResult.AlarmCount++
		policySummary.TotalSeverity.IncreaseBySeverityByAmount(policy.Severity, 1)
	case types.ComplianceResultINFO:
		policySummary.TotalResult.InfoCount++
		policySummary.TotalSeverity.UnknownCount++
	case types.ComplianceResultSKIP:
		policySummary.TotalResult.SkipCount++
		policySummary.TotalSeverity.UnknownCount++
	case types.ComplianceResultERROR:
		policySummary.TotalResult.ErrorCount++
		policySummary.TotalSeverity.IncreaseBySeverityByAmount(policy.Severity, 1)
	}
	b.policySummaries[resource.PolicyID][resource.ConnectionID] = policySummary
}

func (b *BenchmarkSummaryBuilder) extractBenchmarkSummary(benchmark *complianceApi.Benchmark) {
	timeAt := time.Now().Unix()
	connectorTypeMap := make(map[string]map[source.Type]bool)
	if _, ok := b.benchmarkSummaries[benchmark.ID]; !ok {
		b.benchmarkSummaries[benchmark.ID] = make(map[string]types.BenchmarkSummary)
	}
	for _, child := range benchmark.Children {
		if _, ok := b.benchmarkSummaries[child]; !ok {
			childBenchmark, err := b.complianceClient.GetBenchmark(&httpclient.Context{UserRole: authApi.KeibiAdminRole}, child)
			if err != nil {
				b.logger.Error("failed to get benchmark", zap.Error(err))
				continue
			}
			b.extractBenchmarkSummary(childBenchmark)
		}
		childBenchmarkSummaryMap := b.benchmarkSummaries[child]
		for connectionID, childBenchmarkSummary := range childBenchmarkSummaryMap {
			if _, ok := b.benchmarkSummaries[benchmark.ID][connectionID]; !ok {
				b.benchmarkSummaries[benchmark.ID][connectionID] = types.BenchmarkSummary{
					BenchmarkID:    benchmark.ID,
					ConnectionID:   connectionID,
					DescribedAt:    timeAt,
					EvaluatedAt:    timeAt,
					TotalResult:    types.ComplianceResultSummary{},
					ReportType:     types.BenchmarksSummary,
					SummarizeJobId: b.jobID,
					ConnectorTypes: nil,
					Policies:       nil,
				}
			}

			benchmarkSummary := b.benchmarkSummaries[benchmark.ID][connectionID]

			benchmarkSummary.TotalResult.AddComplianceResultSummary(childBenchmarkSummary.TotalResult)
			benchmarkSummary.TotalSeverity.AddSeverityResult(childBenchmarkSummary.TotalSeverity)

			benchmarkSummary.Policies = append(benchmarkSummary.Policies, childBenchmarkSummary.Policies...)

			if _, ok := connectorTypeMap[connectionID]; !ok {
				connectorTypeMap[connectionID] = make(map[source.Type]bool)
			}
			for _, connectorType := range childBenchmarkSummary.ConnectorTypes {
				connectorTypeMap[connectionID][connectorType] = true
			}

			b.benchmarkSummaries[benchmark.ID][connectionID] = benchmarkSummary
		}
	}

	for _, policy := range benchmark.Policies {
		policySummaryMap, ok := b.policySummaries[policy]
		if !ok {
			continue
		}
		for connectionID, policySummary := range policySummaryMap {
			if _, ok := b.benchmarkSummaries[benchmark.ID][connectionID]; !ok {
				b.benchmarkSummaries[benchmark.ID][connectionID] = types.BenchmarkSummary{
					BenchmarkID:    benchmark.ID,
					ConnectionID:   connectionID,
					DescribedAt:    timeAt,
					EvaluatedAt:    timeAt,
					TotalResult:    types.ComplianceResultSummary{},
					ReportType:     types.BenchmarksSummary,
					SummarizeJobId: b.jobID,
					ConnectorTypes: nil,
					Policies:       nil,
				}
			}

			benchmarkSummary := b.benchmarkSummaries[benchmark.ID][connectionID]

			if _, ok := connectorTypeMap[connectionID]; !ok {
				connectorTypeMap[connectionID] = make(map[source.Type]bool)
			}

			connectorTypeMap[connectionID][policySummary.ConnectorType] = true

			benchmarkSummary.TotalResult.AddComplianceResultSummary(policySummary.TotalResult)
			benchmarkSummary.TotalSeverity.AddSeverityResult(policySummary.TotalSeverity)

			benchmarkSummary.Policies = append(benchmarkSummary.Policies, policySummary)

			b.benchmarkSummaries[benchmark.ID][connectionID] = benchmarkSummary
		}
	}

	for connectionID, benchmarkSummary := range b.benchmarkSummaries[benchmark.ID] {
		for _, connectorType := range benchmarkSummary.ConnectorTypes {
			connectorTypeMap[connectionID][connectorType] = true
		}
		benchmarkSummary.ConnectorTypes = nil
		for connectorType := range connectorTypeMap[connectionID] {
			benchmarkSummary.ConnectorTypes = append(benchmarkSummary.ConnectorTypes, connectorType)
		}
		b.benchmarkSummaries[benchmark.ID][connectionID] = benchmarkSummary
	}
}

func (b *BenchmarkSummaryBuilder) Build() []kafka.Doc {
	timeAt := time.Now().Unix()
	var docs []kafka.Doc
	benchmarks, err := b.complianceClient.ListBenchmarks(&httpclient.Context{
		UserRole: authApi.KeibiAdminRole,
	})
	if err != nil {
		b.logger.Error("failed to list benchmarks", zap.Error(err))
		return docs
	}
	for _, benchmark := range benchmarks {
		b.extractBenchmarkSummary(&benchmark)
	}
	for _, benchmarkSummaryMap := range b.benchmarkSummaries {
		for _, benchmarkSummary := range benchmarkSummaryMap {
			docs = append(docs, benchmarkSummary)
			historySummary := benchmarkSummary
			historySummary.ReportType = types.BenchmarksSummaryHistory
			docs = append(docs, historySummary)
		}
	}

	for _, connector := range source.List {
		if connector == source.Nil {
			continue
		}
		for benchmarkId, benchmarkSummaryMap := range b.benchmarkSummaries {
			benchmarkSummary := types.BenchmarkSummary{
				BenchmarkID:    benchmarkId,
				ConnectionID:   connector.String(),
				ConnectorTypes: []source.Type{connector},
				DescribedAt:    timeAt,
				EvaluatedAt:    timeAt,
				Policies:       nil,
				TotalResult:    types.ComplianceResultSummary{},
				TotalSeverity:  types.SeverityResult{},
				ReportType:     types.BenchmarksConnectorSummary,
				SummarizeJobId: b.jobID,
			}
			for _, benchmarkSummaryPerConnection := range benchmarkSummaryMap {
				found := false
				for _, connectorType := range benchmarkSummaryPerConnection.ConnectorTypes {
					if connectorType == connector {
						found = true
						break
					}
				}
				if !found {
					continue
				}

				benchmarkSummary.TotalResult.AddComplianceResultSummary(benchmarkSummaryPerConnection.TotalResult)
				benchmarkSummary.TotalSeverity.AddSeverityResult(benchmarkSummaryPerConnection.TotalSeverity)

				benchmarkSummary.Policies = append(benchmarkSummary.Policies, benchmarkSummaryPerConnection.Policies...)
			}

			docs = append(docs, benchmarkSummary)
			historySummary := benchmarkSummary
			historySummary.ReportType = types.BenchmarksConnectorSummaryHistory
			docs = append(docs, historySummary)
		}
	}

	return docs
}

func (b *BenchmarkSummaryBuilder) Cleanup(summarizeJobID uint) error {
	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must_not": []map[string]any{
					{
						"term": map[string]any{
							"summarize_job_id": summarizeJobID,
						},
					},
				},
				"filter": []map[string]any{
					{
						"terms": map[string]any{
							"report_type": []string{
								string(types.BenchmarksSummary),
								string(types.BenchmarksSummaryHistory),
								string(types.BenchmarksConnectorSummary),
								string(types.BenchmarksConnectorSummaryHistory),
							},
						},
					},
				},
			},
		},
	}
	esClient := b.client.ES()
	resp, err := keibi.DeleteByQuery(context.Background(), esClient, []string{types.BenchmarkSummaryIndex}, query,
		esClient.DeleteByQuery.WithRefresh(true),
		esClient.DeleteByQuery.WithConflicts("proceed"),
	)
	if err != nil {
		b.logger.Error("elasticsearch: delete by query", zap.Error(err))
		return err
	}
	if len(resp.Failures) != 0 {
		body, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		b.logger.Error("elasticsearch: delete by query", zap.String("body", string(body)))
		return fmt.Errorf("elasticsearch: delete by query: %s", string(body))
	}
	return nil
}
