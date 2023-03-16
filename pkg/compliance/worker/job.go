package worker

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	"gitlab.com/keibiengine/keibi-engine/pkg/config"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"
	"time"
)

type Job struct {
	JobID uint

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
	db *db.Database,
	steampipeConn *steampipe.Database,
	vault vault.SourceConfig,
	elasticSearchConfig config.ElasticSearch,
) JobResult {
	result := JobResult{
		JobID:           j.JobID,
		Status:          api.ComplianceReportJobCompleted,
		ReportCreatedAt: time.Now().UnixMilli(),
		Error:           "",
	}

	if err := j.Run(db, steampipeConn, vault, elasticSearchConfig); err != nil {
		result.Error = err.Error()
	}
	result.ReportCreatedAt = time.Now().UnixMilli()
	return result
}

func (j *Job) Run(db *db.Database, steampipeConn *steampipe.Database, vault vault.SourceConfig, elasticSearchConfig config.ElasticSearch) error {
	err := j.PopulateSteampipeConfig(vault, elasticSearchConfig)
	if err != nil {
		return err
	}

	benchmark, err := db.GetBenchmark(j.BenchmarkID)
	if err != nil {
		return err
	}

	for _, policy := range benchmark.Policies {
		res, err := steampipeConn.QueryAll(policy.Query.QueryToExecute)
		if err != nil {
			return err
		}

		err = j.PopulateFindings(db, res)
		if err != nil {
			return err
		}
	}
	return nil
}

func (j *Job) PopulateFindings(db *db.Database, res *steampipe.Result) error {
	return nil
}
