package describe

import (
	"context"
	"encoding/json"

	"github.com/kaytu-io/kaytu-engine/pkg/insight"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

// RunInsightJobResultsConsumer consumes messages from the insightJobResultQueue queue.
// It will update the status of the jobs in the database based on the message.
// It will also update the jobs status that are not completed in certain time to FAILED
func (s *Scheduler) RunInsightJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the InsightJobResultQueue queue")

	s.jq.Consume(context.Background(), "insight-scheduler", insight.InsightStreamName, []string{insight.InsightResultsQueueName}, "insight-scheduler", func(msg jetstream.Msg) {
		var result insight.JobResult

		if err := json.Unmarshal(msg.Data(), &result); err != nil {
			s.logger.Error("Failed to unmarshal InsightJobResult results", zap.Error(err))

			if err := msg.Nak(); err != nil {
				s.logger.Error("Failed nak message", zap.Error(err))
			}

			return
		}

		s.logger.Info("Processing InsightJobResult for Job",
			zap.Uint("jobId", result.JobID),
			zap.String("status", string(result.Status)),
		)

		if err := s.db.UpdateInsightJob(result.JobID, result.Status, result.Error); err != nil {
			s.logger.Error("Failed to update the status of InsightJob",
				zap.Uint("jobId", result.JobID),
				zap.Error(err))

			if err := msg.Nak(); err != nil {
				s.logger.Error("Failed not ack a message", zap.Error(err))
			}

			return
		}

		if err := msg.Ack(); err != nil {
			s.logger.Error("Failed to ack a message", zap.Error(err))
		}
	})

	for {
	}
}
