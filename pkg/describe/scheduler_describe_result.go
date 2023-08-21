package describe

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"

	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"go.uber.org/zap"
)

func (s *Scheduler) RunDescribeJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the JobResults queue")

	msgs, err := s.describeJobResultQueue.Consume()
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
			var result DescribeJobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				ResultsProcessedCount.WithLabelValues("", "failure").Inc()

				s.logger.Error("failed to consume message from describeJobResult", zap.Error(err))
				err = msg.Nack(false, false)
				if err != nil {
					s.logger.Error("failure while sending nack for message", zap.Error(err))
				}
				continue
			}

			s.logger.Info("Processing JobResult for Job",
				zap.Uint("jobId", result.JobID),
				zap.String("status", string(result.Status)),
			)

			if s.DoDeleteOldResources {
				if err := s.cleanupOldResources(result); err != nil {
					ResultsProcessedCount.WithLabelValues(string(result.DescribeJob.SourceType), "failure").Inc()
					s.logger.Error("failed to cleanupOldResources", zap.Error(err))
					err = msg.Nack(false, true)
					if err != nil {
						s.logger.Error("failure while sending nack for message", zap.Error(err))
					}
					continue
				}
			}

			errStr := strings.ReplaceAll(result.Error, "\x00", "")
			errCodeStr := strings.ReplaceAll(result.ErrorCode, "\x00", "")
			if err := s.db.UpdateDescribeResourceJobStatus(result.JobID, result.Status, errStr, errCodeStr, int64(len(result.DescribedResourceIDs))); err != nil {
				ResultsProcessedCount.WithLabelValues(string(result.DescribeJob.SourceType), "failure").Inc()
				s.logger.Error("failed to UpdateDescribeResourceJobStatus", zap.Error(err))
				err = msg.Nack(false, true)
				if err != nil {
					s.logger.Error("failure while sending nack for message", zap.Error(err))
				}
				continue
			}

			ResultsProcessedCount.WithLabelValues(string(result.DescribeJob.SourceType), "successful").Inc()
			if err := msg.Ack(false); err != nil {
				s.logger.Error("failure while sending ack for message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateDescribeResourceJobsTimedOut(s.describeTimeoutHours)
			DescribeResourceJobsCount.WithLabelValues("failure").Inc()
			if err != nil {
				s.logger.Error("failed to update timed out DescribeResourceJobs", zap.Error(err))
			}
		}
	}
}

func (s *Scheduler) cleanupOldResources(res DescribeJobResult) error {
	var searchAfter []any

	deletedCount := 0
	s.logger.Info("starting to delete old resources",
		zap.Uint("jobId", res.JobID),
		zap.String("connection_id", res.DescribeJob.SourceID),
		zap.String("resource_type", res.DescribeJob.ResourceType),
	)
	for {
		esResp, err := es.GetResourceIDsForAccountResourceTypeFromES(
			s.es,
			res.DescribeJob.SourceID,
			res.DescribeJob.ResourceType,
			searchAfter,
			1000)
		if err != nil {
			return err
		}

		if len(esResp.Hits.Hits) == 0 {
			break
		}
		var msgs []*confluent_kafka.Message
		for _, hit := range esResp.Hits.Hits {
			searchAfter = hit.Sort
			esResourceID := hit.Source.ResourceID

			exists := false
			for _, describedResourceID := range res.DescribedResourceIDs {
				if esResourceID == describedResourceID {
					exists = true
					break
				}
			}

			if !exists {
				OldResourcesDeletedCount.WithLabelValues(string(res.DescribeJob.SourceType)).Inc()
				resource := es.Resource{
					ID:           esResourceID,
					ResourceType: res.DescribeJob.ResourceType,
					SourceType:   res.DescribeJob.SourceType,
				}
				keys, idx := resource.KeysAndIndex()
				msg := kafka.Msg(kafka.HashOf(keys...), nil, idx, s.kafkaResourcesTopic, confluent_kafka.PartitionAny)
				msgs = append(msgs, msg)

				lookupResource := es.LookupResource{
					ResourceID:   esResourceID,
					ResourceType: res.DescribeJob.ResourceType,
					SourceType:   res.DescribeJob.SourceType,
				}
				lookUpKeys, lookUpIdx := lookupResource.KeysAndIndex()
				msg = kafka.Msg(kafka.HashOf(lookUpKeys...), nil, lookUpIdx, s.kafkaResourcesTopic, confluent_kafka.PartitionAny)
				msgs = append(msgs, msg)
				if err != nil {
					return err
				}
			}
		}
		_, err = kafka.SyncSend(s.logger, s.kafkaProducer, msgs)
		if err != nil {
			s.logger.Error("failed to send delete message to kafka",
				zap.Uint("jobId", res.JobID),
				zap.String("connection_id", res.DescribeJob.SourceID),
				zap.String("resource_type", res.DescribeJob.ResourceType),
				zap.Error(err))
			return err
		}
		deletedCount += len(msgs)
	}
	s.logger.Info("deleted old resources",
		zap.Uint("jobId", res.JobID),
		zap.String("connection_id", res.DescribeJob.SourceID),
		zap.String("resource_type", res.DescribeJob.ResourceType),
		zap.Int("deleted_count", deletedCount))

	return nil
}

func (s *Scheduler) cleanupDeletedConnectionResources(connectionId string) error {
	var searchAfter []interface{}

	for {
		esResp, err := es.GetResourceIDsForAccountFromES(s.es, connectionId, searchAfter, 1000)
		if err != nil {
			return err
		}

		if len(esResp.Hits.Hits) == 0 {
			break
		}
		var msgs []*confluent_kafka.Message
		for _, hit := range esResp.Hits.Hits {
			searchAfter = hit.Sort
			esResourceID := hit.Source.ResourceID

			resource := es.Resource{
				ID:           esResourceID,
				ResourceType: strings.ToLower(hit.Source.ResourceType),
				SourceType:   hit.Source.SourceType,
			}
			keys, idx := resource.KeysAndIndex()
			key := kafka.HashOf(keys...)
			msg := kafka.Msg(key, nil, idx, s.kafkaResourcesTopic, confluent_kafka.PartitionAny)
			msgs = append(msgs, msg)

			lookupResource := es.LookupResource{
				ResourceID:   esResourceID,
				ResourceType: strings.ToLower(hit.Source.ResourceType),
				SourceType:   hit.Source.SourceType,
			}
			keys, idx = lookupResource.KeysAndIndex()
			key = kafka.HashOf(keys...)
			msg = kafka.Msg(key, nil, idx, s.kafkaResourcesTopic, confluent_kafka.PartitionAny)
			msgs = append(msgs, msg)
			if err != nil {
				return err
			}
		}
		_, err = kafka.SyncSend(s.logger, s.kafkaProducer, msgs)
		if err != nil {
			s.logger.Error("failed to send delete message to kafka", zap.Error(err))
			return err
		}
	}

	return nil
}

func (s *Scheduler) RunDescribeJobCompletionUpdater() {
	t := time.NewTicker(JobCompletionInterval)
	defer t.Stop()

	for ; ; <-t.C {
		results, err := s.db.QueryInProgressDescribedSourceJobGroupByDescribeResourceJobStatus()
		if err != nil {
			s.logger.Error("Failed to find DescribeSourceJobs", zap.Error(err))
			continue
		}

		jobIDToStatus := make(map[uint]map[api.DescribeResourceJobStatus]int)
		jobStatus := make(map[uint]api.DescribeSourceJobStatus)
		for _, v := range results {
			if _, ok := jobIDToStatus[v.DescribeSourceJobID]; !ok {
				jobIDToStatus[v.DescribeSourceJobID] = map[api.DescribeResourceJobStatus]int{
					api.DescribeResourceJobCreated:   0,
					api.DescribeResourceJobQueued:    0,
					api.DescribeResourceJobTimeout:   0,
					api.DescribeResourceJobFailed:    0,
					api.DescribeResourceJobSucceeded: 0,
				}
			}

			jobStatus[v.DescribeSourceJobID] = v.DescribeSourceStatus
			jobIDToStatus[v.DescribeSourceJobID][v.DescribeResourceJobStatus] = v.DescribeResourceJobCount
		}

		for id, status := range jobIDToStatus {
			// If any CREATED or QUEUED, job is still in progress
			if status[api.DescribeResourceJobCreated] > 0 ||
				status[api.DescribeResourceJobQueued] > 0 ||
				status[api.DescribeResourceJobInProgress] > 0 {
				if jobStatus[id] == api.DescribeSourceJobCreated {
					err := s.db.UpdateDescribeSourceJob(id, api.DescribeSourceJobInProgress)
					if err != nil {
						s.logger.Error("Failed to update DescribeSourceJob status\n",
							zap.Uint("jobId", id),
							zap.String("status", string(api.DescribeSourceJobInProgress)),
							zap.Error(err),
						)
					}
				}
				continue
			}

			// If any FAILURE, job is completed with failure
			if status[api.DescribeResourceJobFailed] > 0 || status[api.DescribeResourceJobTimeout] > 0 {
				err := s.db.UpdateDescribeSourceJob(id, api.DescribeSourceJobCompletedWithFailure)
				if err != nil {
					s.logger.Error("Failed to update DescribeSourceJob status\n",
						zap.Uint("jobId", id),
						zap.String("status", string(api.DescribeSourceJobCompletedWithFailure)),
						zap.Error(err),
					)
				}
				continue
			}

			// If the rest is SUCCEEDED, job has completed with no failure
			if status[api.DescribeResourceJobSucceeded] > 0 {
				err := s.db.UpdateDescribeSourceJob(id, api.DescribeSourceJobCompleted)
				if err != nil {
					s.logger.Error("Failed to update DescribeSourceJob status\n",
						zap.Uint("jobId", id),
						zap.String("status", string(api.DescribeSourceJobCompleted)),
						zap.Error(err),
					)
				}
				continue
			}
		}
	}
}
