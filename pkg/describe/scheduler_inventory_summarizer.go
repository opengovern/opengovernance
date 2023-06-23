package describe

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer"
	summarizerapi "github.com/kaytu-io/kaytu-engine/pkg/summarizer/api"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

func (s Scheduler) RunMustSummerizeJobScheduler() {
	s.logger.Info("Scheduling must summerize jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		lastJob, err := s.db.FetchLastSummarizerJob()
		if err != nil {
			s.logger.Error("Failed to find the last job to check for MustSummerizeJob", zap.Error(err))
			continue
		}
		if lastJob == nil || lastJob.CreatedAt.Add(time.Duration(s.mustSummarizeIntervalHours)*time.Hour).Before(time.Now()) {
			err := s.scheduleMustSummarizerJob(nil)
			if err != nil {
				s.logger.Error("failure on scheduleMustSummarizerJob", zap.Error(err))
			}
		}
	}
}

func (s Scheduler) scheduleMustSummarizerJob(scheduleJobID *uint) error {
	ongoingJobs, err := s.db.GetOngoingSummarizerJobsByType(summarizer.JobType_ResourceMustSummarizer)
	if err != nil {
		SummarizerJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to get ongoing SummarizerJobs",
			zap.Error(err),
		)
		return err
	}
	if len(ongoingJobs) > 0 {
		s.logger.Info("There is ongoing MustSummarizerJob skipping this schedule")
		return fmt.Errorf("there is ongoing MustSummarizerJob skipping this schedule")
	}

	job := newMustSummarizerJob(scheduleJobID)
	err = s.db.AddSummarizerJob(&job)
	if err != nil {
		SummarizerJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to create SummarizerJob",
			zap.Uint("jobId", job.ID),
			zap.Error(err),
		)
		return err
	}

	err = enqueueMustSummarizerJobs(s.db, s.summarizerJobQueue, job)
	if err != nil {
		SummarizerJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to enqueue SummarizerJob",
			zap.Uint("jobId", job.ID),
			zap.Error(err),
		)
		job.Status = summarizerapi.SummarizerJobFailed
		err = s.db.UpdateSummarizerJobStatus(job)
		if err != nil {
			s.logger.Error("Failed to update SummarizerJob status",
				zap.Uint("jobId", job.ID),
				zap.Error(err),
			)
		}
		return err
	}

	return nil
}

func enqueueMustSummarizerJobs(db Database, q queue.Interface, job SummarizerJob) error {
	var lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint

	lastDay, err := db.GetOldCompletedScheduleJob(1)
	if err != nil {
		return err
	}
	if lastDay != nil {
		lastDayJobID = lastDay.ID
	}
	lastWeek, err := db.GetOldCompletedScheduleJob(7)
	if err != nil {
		return err
	}
	if lastWeek != nil {
		lastWeekJobID = lastWeek.ID
	}
	lastQuarter, err := db.GetOldCompletedScheduleJob(93)
	if err != nil {
		return err
	}
	if lastQuarter != nil {
		lastQuarterJobID = lastQuarter.ID
	}
	lastYear, err := db.GetOldCompletedScheduleJob(428)
	if err != nil {
		return err
	}
	if lastYear != nil {
		lastYearJobID = lastYear.ID
	}

	if err := q.Publish(summarizer.ResourceJob{
		JobID:                    job.ID,
		LastDayScheduleJobID:     lastDayJobID,
		LastWeekScheduleJobID:    lastWeekJobID,
		LastQuarterScheduleJobID: lastQuarterJobID,
		LastYearScheduleJobID:    lastYearJobID,
		JobType:                  summarizer.JobType_ResourceMustSummarizer,
	}); err != nil {
		return err
	}

	return nil
}

func newMustSummarizerJob(scheduleJobID *uint) SummarizerJob {
	return SummarizerJob{
		Model:          gorm.Model{},
		Status:         summarizerapi.SummarizerJobInProgress,
		ScheduleJobID:  scheduleJobID,
		JobType:        summarizer.JobType_ResourceMustSummarizer,
		FailureMessage: "",
	}
}

// RunSummarizerJobResultsConsumer consumes messages from the summarizerJobResultQueue queue.
// It will update the status of the jobs in the database based on the message.
// It will also update the jobs status that are not completed in certain time to FAILED
func (s *Scheduler) RunSummarizerJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the summarizerJobResultQueue queue")

	msgs, err := s.summarizerJobResultQueue.Consume()
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

			var result summarizer.ResourceJobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				s.logger.Error("Failed to unmarshal SummarizerJobResult results", zap.Error(err))
				err = msg.Nack(false, false)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			if result.JobType == "" || result.JobType == summarizer.JobType_ResourceSummarizer || result.JobType == summarizer.JobType_ResourceMustSummarizer {
				s.logger.Info("Processing SummarizerJobResult for Job",
					zap.Uint("jobId", result.JobID),
					zap.String("status", string(result.Status)),
				)
				err := s.db.UpdateSummarizerJob(result.JobID, result.Status, result.Error)
				if err != nil {
					s.logger.Error("Failed to update the status of SummarizerJob",
						zap.Uint("jobId", result.JobID),
						zap.Error(err))
					err = msg.Nack(false, true)
					if err != nil {
						s.logger.Error("Failed nacking message", zap.Error(err))
					}
					continue
				}
			} else {
				var result summarizer.ComplianceJobResult
				if err := json.Unmarshal(msg.Body, &result); err != nil {
					s.logger.Error("Failed to unmarshal SummarizerJobResult results", zap.Error(err))
					err = msg.Nack(false, false)
					if err != nil {
						s.logger.Error("Failed nacking message", zap.Error(err))
					}
					continue
				}

				s.logger.Info("Processing SummarizerJobResult for Job",
					zap.Uint("jobId", result.JobID),
					zap.String("status", string(result.Status)),
				)
				err := s.db.UpdateSummarizerJob(result.JobID, result.Status, result.Error)
				if err != nil {
					s.logger.Error("Failed to update the status of SummarizerJob",
						zap.Uint("jobId", result.JobID),
						zap.Error(err))
					err = msg.Nack(false, true)
					if err != nil {
						s.logger.Error("Failed nacking message", zap.Error(err))
					}
					continue
				}
			}

			if err := msg.Ack(false); err != nil {
				s.logger.Error("Failed acking message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateSummarizerJobsTimedOut(s.summarizerIntervalHours)
			if err != nil {
				s.logger.Error("Failed to update timed out SummarizerJob", zap.Error(err))
			}
		}
	}
}
