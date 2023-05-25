package describe

import (
	"encoding/json"
	"fmt"
	"gitlab.com/keibiengine/keibi-engine/pkg/insight"
	"time"

	"go.uber.org/zap"
)

// RunInsightJobResultsConsumer consumes messages from the insightJobResultQueue queue.
// It will update the status of the jobs in the database based on the message.
// It will also update the jobs status that are not completed in certain time to FAILED
func (s *Scheduler) RunInsightJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the InsightJobResultQueue queue")

	msgs, err := s.insightJobResultQueue.Consume()
	if err != nil {
		return err
	}

	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("tasks channel is closed")
			}

			var result insight.JobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				s.logger.Error("Failed to unmarshal InsightJobResult results", zap.Error(err))
				err = msg.Nack(false, false)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			s.logger.Info("Processing InsightJobResult for Job",
				zap.Uint("jobId", result.JobID),
				zap.String("status", string(result.Status)),
			)
			err := s.db.UpdateInsightJob(result.JobID, result.Status, result.Error)
			if err != nil {
				s.logger.Error("Failed to update the status of InsightJob",
					zap.Uint("jobId", result.JobID),
					zap.Error(err))
				err = msg.Nack(false, true)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			if err := msg.Ack(false); err != nil {
				s.logger.Error("Failed acking message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateInsightJobsTimedOut(s.insightIntervalHours)
			if err != nil {
				s.logger.Error("Failed to update timed out InsightJob", zap.Error(err))
			}
		}
	}
}
