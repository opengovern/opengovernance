package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kaytu-io/open-governance/pkg/compliance/summarizer"
	types2 "github.com/kaytu-io/open-governance/pkg/compliance/summarizer/types"
	"github.com/kaytu-io/open-governance/pkg/describe/db/model"
	"github.com/kaytu-io/open-governance/pkg/types"
	"go.uber.org/zap"
)

const SummarizerSchedulingInterval = 1 * time.Minute

type SankDocumentCountResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		}
	}
}

func (s *JobScheduler) getSankDocumentCountBenchmark(ctx context.Context, benchmarkId string, parentJobID uint) (int, error) {
	request := make(map[string]any)
	filters := make([]map[string]any, 0)
	filters = append(filters, map[string]any{
		"term": map[string]any{
			"benchmarkID": benchmarkId,
		},
	})
	filters = append(filters, map[string]any{
		"term": map[string]any{
			"parentComplianceJobID": parentJobID,
		},
	})
	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	request["size"] = 0

	query, err := json.Marshal(request)
	if err != nil {
		s.logger.Error("failed to marshal request", zap.Error(err))
		return 0, err
	}

	s.logger.Info("GetSankDocumentCountBenchmark", zap.String("benchmarkId", benchmarkId), zap.String("query", string(query)))

	sankDocumentCountResponse := SankDocumentCountResponse{}
	err = s.esClient.SearchWithTrackTotalHits(
		ctx, types.FindingsIndex,
		string(query),
		nil,
		&sankDocumentCountResponse, true,
	)
	if err != nil {
		s.logger.Error("failed to get sank document count", zap.Error(err), zap.String("benchmarkId", benchmarkId))
		return 0, err
	}

	return sankDocumentCountResponse.Hits.Total.Value, nil
}

func (s *JobScheduler) runSummarizer(ctx context.Context, manuals bool) error {
	s.logger.Info("checking for benchmarks to summarize")

	err := s.db.SetJobToRunnersInProgress()
	if err != nil {
		s.logger.Error("failed to set jobs to runners in progress", zap.Error(err))
		return err
	}

	err = s.db.UpdateComplianceJobsTimedOut(24)
	if err != nil {
		s.logger.Error("failed to update compliance jobs timed out", zap.Error(err))
		return err
	}

	jobs, err := s.db.ListJobsWithRunnersCompleted(manuals)
	if err != nil {
		s.logger.Error("failed to list jobs with runners completed", zap.Error(err))
		return err
	}
	if len(jobs) == 0 {
		s.logger.Info("no jobs with runners completed, skipping this summarizer scheduling")
	}
	for _, job := range jobs {
		sankDocCount, err := s.getSankDocumentCountBenchmark(ctx, job.BenchmarkID, job.ID)
		if err != nil {
			s.logger.Error("failed to get sank document count", zap.Error(err), zap.String("benchmarkId", job.BenchmarkID))
			return err
		}
		totalDocCount, err := s.db.FetchTotalFindingCountForComplianceJob(job.ID)
		if err != nil {
			s.logger.Error("failed to get total document count", zap.Error(err), zap.String("benchmarkId", job.BenchmarkID))
			return err
		}

		lastUpdatedRunner, err := s.db.GetLastUpdatedRunnerForParent(job.ID)
		if err != nil {
			s.logger.Error("failed to get last updated runner", zap.Error(err), zap.String("benchmarkId", job.BenchmarkID))
			return err
		}

		if sankDocCount != totalDocCount {
			// do not summarize if all docs are not sank
			// do not summarize if either less than 90% of the docs are sank or last job update is in less than an hour ago
			s.logger.Info("waiting for documents to sink",
				zap.String("benchmarkId", job.BenchmarkID),
				zap.Int("sankDocCount", sankDocCount),
				zap.Int("totalDocCount", totalDocCount),
				zap.Time("lastUpdatedRunner", lastUpdatedRunner.UpdatedAt),
			)
			if job.Status != model.ComplianceJobSinkInProgress {
				err = s.db.UpdateComplianceJob(job.ID, model.ComplianceJobSinkInProgress, "")
				if err != nil {
					s.logger.Error("failed to update compliance job status", zap.Error(err), zap.String("benchmarkId", job.BenchmarkID))
					return err
				}
				continue
			} else if time.Now().Add(-10 * time.Minute).Before(job.UpdatedAt) {
				continue
			}
		}
		s.logger.Info("documents are sank, creating summarizer", zap.String("benchmarkId", job.BenchmarkID), zap.Int("sankDocCount", sankDocCount), zap.Int("totalDocCount", totalDocCount))

		err = s.CreateSummarizer(job.BenchmarkID, &job.ID, job.TriggerType)
		if err != nil {
			s.logger.Error("failed to create summarizer", zap.Error(err), zap.String("benchmarkId", job.BenchmarkID))
			return err
		}
	}

	createds, err := s.db.FetchCreatedSummarizers(manuals)
	if err != nil {
		s.logger.Error("failed to fetch created summarizers", zap.Error(err))
		return err
	}

	for _, job := range createds {
		err = s.triggerSummarizer(ctx, job)
		if err != nil {
			s.logger.Error("failed to trigger summarizer", zap.Error(err), zap.String("benchmarkId", job.BenchmarkID))
			return err
		}
	}

	jobs, err = s.db.ListJobsToFinish()
	for _, job := range jobs {
		err = s.finishComplianceJob(job)
		if err != nil {
			s.logger.Error("failed to finish compliance job", zap.Error(err), zap.String("benchmarkId", job.BenchmarkID))
			return err
		}
	}

	err = s.db.RetryFailedSummarizers()
	if err != nil {
		s.logger.Error("failed to retry failed runners", zap.Error(err))
		return err
	}

	return nil
}

func (s *JobScheduler) finishComplianceJob(job model.ComplianceJob) error {
	failedRunners, err := s.db.ListFailedRunnersWithParentID(job.ID)
	if err != nil {
		return err
	}

	if len(failedRunners) > 0 {
		builder := strings.Builder{}
		builder.WriteString(fmt.Sprintf("%d runners failed: [", len(failedRunners)))
		for i, runner := range failedRunners {
			builder.WriteString(fmt.Sprintf("%s", runner.FailureMessage))
			if i != len(failedRunners)-1 {
				builder.WriteString(", ")
			}
		}
		builder.WriteString("]")

		return s.db.UpdateComplianceJob(job.ID, model.ComplianceJobFailed, builder.String())
	}

	failedSummarizers, err := s.db.ListFailedSummarizersWithParentID(job.ID)
	if err != nil {
		return err
	}

	if len(failedSummarizers) > 0 {
		builder := strings.Builder{}
		builder.WriteString(fmt.Sprintf("%d summarizers failed: [", len(failedSummarizers)))
		for i, summarizer := range failedSummarizers {
			builder.WriteString(fmt.Sprintf("%s", summarizer.FailureMessage))
			if i != len(failedSummarizers)-1 {
				builder.WriteString(", ")
			}
		}
		builder.WriteString("]")
		return s.db.UpdateComplianceJob(job.ID, model.ComplianceJobFailed, builder.String())
	}

	return s.db.UpdateComplianceJob(job.ID, model.ComplianceJobSucceeded, "")
}

func (s *JobScheduler) CreateSummarizer(benchmarkId string, jobId *uint, triggerType model.ComplianceTriggerType) error {
	// run summarizer
	dbModel := model.ComplianceSummarizer{
		BenchmarkID: benchmarkId,
		StartedAt:   time.Now(),
		Status:      summarizer.ComplianceSummarizerCreated,
		TriggerType: triggerType,
	}
	if jobId != nil {
		dbModel.ParentJobID = *jobId
	}
	err := s.db.CreateSummarizerJob(&dbModel)
	if err != nil {
		return err
	}
	if jobId != nil {
		return s.db.UpdateComplianceJob(*jobId, model.ComplianceJobSummarizerInProgress, "")
	}
	return nil
}

func (s *JobScheduler) triggerSummarizer(ctx context.Context, job model.ComplianceSummarizer) error {
	summarizerJob := types2.Job{
		ID:          job.ID,
		RetryCount:  job.RetryCount,
		BenchmarkID: job.BenchmarkID,
		CreatedAt:   job.CreatedAt,
	}
	jobJson, err := json.Marshal(summarizerJob)
	if err != nil {
		_ = s.db.UpdateSummarizerJob(job.ID, summarizer.ComplianceSummarizerFailed, job.CreatedAt, err.Error())
		return err
	}

	topic := summarizer.JobQueueTopic
	if job.TriggerType == model.ComplianceTriggerTypeManual {
		topic = summarizer.JobQueueTopicManuals
	}
	if _, err := s.jq.Produce(ctx, topic, jobJson, fmt.Sprintf("job-%d-%d", job.ID, job.RetryCount)); err != nil {
		_ = s.db.UpdateSummarizerJob(job.ID, summarizer.ComplianceSummarizerFailed, job.CreatedAt, err.Error())
		return err
	}

	return s.db.UpdateSummarizerJob(job.ID, summarizer.ComplianceSummarizerInProgress, job.CreatedAt, "")
}
