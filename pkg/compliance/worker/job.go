package worker

import (
	"fmt"
	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/client"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/config"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
	"time"
)

type Job struct {
	JobID         uint
	ScheduleJobID uint
	DescribedAt   int64
	EvaluatedAt   int64

	ConnectionID string
	BenchmarkID  string

	ConfigReg string
	Connector source.Type
}

type JobResult struct {
	JobID           uint
	Status          api.ComplianceReportJobStatus
	ReportCreatedAt int64
	Error           string
}

func (j *Job) Do(
	complianceClient client.ComplianceServiceClient,
	vault vault.SourceConfig,
	elasticSearchConfig config.ElasticSearch,
	kfkProducer sarama.SyncProducer,
	kfkTopic string,
	logger *zap.Logger,
) JobResult {
	result := JobResult{
		JobID:           j.JobID,
		Status:          api.ComplianceReportJobCompleted,
		ReportCreatedAt: time.Now().UnixMilli(),
		Error:           "",
	}

	if err := j.Run(complianceClient, vault, elasticSearchConfig, kfkProducer, kfkTopic, logger); err != nil {
		result.Error = err.Error()
		result.Status = api.ComplianceReportJobCompletedWithFailure
	}
	result.ReportCreatedAt = time.Now().UnixMilli()
	return result
}

func (j *Job) Run(complianceClient client.ComplianceServiceClient, vault vault.SourceConfig,
	elasticSearchConfig config.ElasticSearch, kfkProducer sarama.SyncProducer, kfkTopic string, logger *zap.Logger) error {
	err := j.PopulateSteampipeConfig(vault, elasticSearchConfig)
	if err != nil {
		return err
	}

	steampipeConn, err := steampipe.NewSteampipeDatabase(steampipe.Option{
		Host: "localhost",
		Port: "9193",
		User: "steampipe",
		Pass: "abcd",
		Db:   "steampipe",
	})
	if err != nil {
		return err
	}

	ctx := &httpclient.Context{
		UserRole:       "",
		UserID:         "",
		WorkspaceName:  "",
		WorkspaceID:    "",
		MaxUsers:       0,
		MaxConnections: 0,
		MaxResources:   0,
	}
	benchmark, err := complianceClient.GetBenchmark(ctx, j.BenchmarkID)
	if err != nil {
		return err
	}

	for _, policyID := range benchmark.Policies {
		policy, err := complianceClient.GetPolicy(ctx, policyID)
		if err != nil {
			return err
		}

		if policy.QueryID == nil {
			continue
		}

		query, err := complianceClient.GetQuery(ctx, *policy.QueryID)
		if err != nil {
			return err
		}

		res, err := steampipeConn.QueryAll(query.QueryToExecute)
		if err != nil {
			return err
		}

		findings, err := j.ExtractFindings(benchmark, policy, query, res)
		if err != nil {
			return err
		}

		var docs []kafka.Doc
		for _, finding := range findings {
			docs = append(docs, finding)
		}

		err = kafka.DoSend(kfkProducer, kfkTopic, docs, logger)
		if err != nil {
			return err
		}
	}
	return nil
}

func (j *Job) ExtractFindings(benchmark *api.Benchmark, policy *api.Policy, query *api.Query, res *steampipe.Result) ([]es.Finding, error) {
	var findings []es.Finding
	for _, record := range res.Data {
		if len(record) != len(res.Headers) {
			return nil, fmt.Errorf("invalid record length, record=%d headers=%d", len(record), len(res.Headers))
		}
		recordValue := map[string]interface{}{}
		for idx, header := range res.Headers {
			value := record[idx]
			recordValue[header] = value
		}

		resourceID := recordValue["resource"].(string)
		resourceName := recordValue["name"].(string)
		resourceType := recordValue["resourceType"].(string)
		resourceLocation := recordValue["location"].(string)
		reason := recordValue["reason"].(string)
		status := recordValue["status"].(string)

		findings = append(findings, es.Finding{
			ID:               fmt.Sprintf("%s-%s-%d", resourceID, policy.ID, j.ScheduleJobID),
			ComplianceJobID:  j.JobID,
			ScheduleJobID:    j.ScheduleJobID,
			ResourceID:       resourceID,
			ResourceName:     resourceName,
			ResourceType:     resourceType,
			ServiceName:      cloudservice.ServiceNameByResourceType(resourceType),
			Category:         cloudservice.CategoryByResourceType(resourceType),
			ResourceLocation: resourceLocation,
			Reason:           reason,
			Status:           types.ComplianceResult(status),
			DescribedAt:      j.DescribedAt,
			EvaluatedAt:      j.EvaluatedAt,
			ConnectionID:     j.ConnectionID,
			Connector:        j.Connector,
			BenchmarkID:      j.BenchmarkID,
			PolicyID:         policy.ID,
			PolicySeverity:   policy.Severity,
		})
	}
	return findings, nil
}
