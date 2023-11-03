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
		err = s.triggerSummarizer(job)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *JobScheduler) triggerSummarizer(job model.ComplianceJob) error {
	// run summarizer
	summarizerJob := summarizer.Job{
		ID:          job.ID,
		BenchmarkID: job.BenchmarkID,
		CreatedAt:   job.CreatedAt,
	}

	jobJson, err := json.Marshal(summarizerJob)
	if err != nil {
		_ = s.db.UpdateComplianceJob(job.ID, model.ComplianceJobFailed, err.Error())
		return err
	}

	msg := kafka2.Msg(fmt.Sprintf("job-%d", job.ID), jobJson, "", summarizer.JobQueue, kafka.PartitionAny)
	_, err = kafka2.SyncSend(s.logger, s.kafkaProducer, []*kafka.Message{msg}, nil)
	if err != nil {
		_ = s.db.UpdateComplianceJob(job.ID, model.ComplianceJobFailed, err.Error())
		return err
	}

	_ = s.db.UpdateComplianceJob(job.ID, model.ComplianceJobSummarizerInProgress, "")
	return nil
}
