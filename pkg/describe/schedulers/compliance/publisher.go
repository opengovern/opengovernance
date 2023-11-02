package compliance

import (
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	kafka2 "github.com/kaytu-io/kaytu-util/pkg/kafka"
)

func (s *JobScheduler) runPublisher() error {
	runners, err := s.db.FetchCreatedRunners()
	if err != nil {
		return err
	}

	for _, job := range runners {
		jobJson, err := json.Marshal(job)
		if err != nil {
			_ = s.db.UpdateRunnerJob(job.ID, runner.ComplianceRunnerFailed, err.Error())
			return err
		}

		msg := kafka2.Msg(fmt.Sprintf("job-%d", job.ID), jobJson, "", runner.JobQueue, kafka.PartitionAny)
		_, err = kafka2.SyncSend(s.logger, s.kafkaProducer, []*kafka.Message{msg}, nil)
		if err != nil {
			_ = s.db.UpdateRunnerJob(job.ID, runner.ComplianceRunnerFailed, err.Error())
			return err
		}

		_ = s.db.UpdateRunnerJob(job.ID, runner.ComplianceRunnerInProgress, err.Error())
	}

	err = s.db.RetryFailedRunners()
	if err != nil {
		return err
	}

	return nil
}
