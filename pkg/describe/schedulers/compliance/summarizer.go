package compliance

import (
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	kafka2 "github.com/kaytu-io/kaytu-util/pkg/kafka"
	"time"
)

const SummarizerSchedulingInterval = 5 * time.Minute

func (s *JobScheduler) runSummarizer() error {
	err := s.db.SetJobToRunnersInProgress()
	if err != nil {
		return err
	}

	jobs, err := s.db.ListJobsToSummarize()
	for _, job := range jobs {
		err = s.createSummarizer(job)
		if err != nil {
			return err
		}
	}

	createds, err := s.db.FetchCreatedSummarizers()
	if err != nil {
		return err
	}

	for _, job := range createds {
		err = s.triggerSummarizer(job)
		if err != nil {
			return err
		}
	}

	jobs, err = s.db.ListJobsToFinish()
	for _, job := range jobs {
		err = s.finishComplianceJob(job)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *JobScheduler) finishComplianceJob(job model.ComplianceJob) error {
	failedRunners, err := s.db.ListFailedRunnersWithParentID(job.ID)
	if err != nil {
		return err
	}

	if len(failedRunners) > 0 {
		return s.db.UpdateComplianceJob(job.ID, model.ComplianceJobFailed, fmt.Sprintf("%d runners failed", len(failedRunners)))
	}

	failedSummarizers, err := s.db.ListFailedSummarizersWithParentID(job.ID)
	if err != nil {
		return err
	}

	if len(failedSummarizers) > 0 {
		return s.db.UpdateComplianceJob(job.ID, model.ComplianceJobFailed, fmt.Sprintf("%d summarizers failed", len(failedSummarizers)))
	}

	return s.db.UpdateComplianceJob(job.ID, model.ComplianceJobSucceeded, "")
}

func (s *JobScheduler) createSummarizer(job model.ComplianceJob) error {
	// run summarizer
	dbModel := model.ComplianceSummarizer{
		BenchmarkID: job.BenchmarkID,
		ParentJobID: job.ID,
		StartedAt:   time.Now(),
		Status:      summarizer.ComplianceSummarizerCreated,
	}
	err := s.db.CreateSummarizerJob(&dbModel)
	if err != nil {
		return err
	}

	return s.db.UpdateComplianceJob(job.ID, model.ComplianceJobSummarizerInProgress, "")
}

func (s *JobScheduler) triggerSummarizer(job model.ComplianceSummarizer) error {
	summarizerJob := summarizer.Job{
		ID:          job.ID,
		BenchmarkID: job.BenchmarkID,
		CreatedAt:   job.CreatedAt,
	}
	jobJson, err := json.Marshal(summarizerJob)
	if err != nil {
		_ = s.db.UpdateSummarizerJob(job.ID, summarizer.ComplianceSummarizerFailed, job.CreatedAt, err.Error())
		return err
	}

	msg := kafka2.Msg(fmt.Sprintf("job-%d", job.ID), jobJson, "", summarizer.JobQueue, kafka.PartitionAny)
	_, err = kafka2.SyncSend(s.logger, s.kafkaProducer, []*kafka.Message{msg}, nil)
	if err != nil {
		_ = s.db.UpdateSummarizerJob(job.ID, summarizer.ComplianceSummarizerFailed, job.CreatedAt, err.Error())
		return err
	}

	return s.db.UpdateSummarizerJob(job.ID, summarizer.ComplianceSummarizerInProgress, job.CreatedAt, err.Error())
}
