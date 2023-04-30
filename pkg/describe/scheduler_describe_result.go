package describe

import (
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
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
				zap.Strings("resourceIDs", result.DescribedResourceIDs),
			)

			//if err := s.cleanupOldResources(result); err != nil {
			//	s.logger.Error("failed to cleanupOldResources", zap.Error(err))
			//	err = msg.Nack(false, true)
			//	if err != nil {
			//		s.logger.Error("failure while sending nack for message", zap.Error(err))
			//	}
			//	continue
			//}

			if err := s.db.UpdateDescribeResourceJobStatus(result.JobID, result.Status, result.Error); err != nil {
				s.logger.Error("failed to UpdateDescribeResourceJobStatus", zap.Error(err))
				err = msg.Nack(false, true)
				if err != nil {
					s.logger.Error("failure while sending nack for message", zap.Error(err))
				}
				continue
			}

			if err := msg.Ack(false); err != nil {
				s.logger.Error("failure while sending ack for message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateDescribeResourceJobsTimedOut(s.describeTimeoutHours)
			if err != nil {
				s.logger.Error("failed to update timed out DescribeResourceJobs", zap.Error(err))
			}
		}
	}
}

func (s *Scheduler) cleanupOldResources(res DescribeJobResult) error {
	var kafkaMsgs []*sarama.ProducerMessage
	var esResourceIDs []string
	var searchAfter []interface{}

	for {
		esResp, err := es.GetResourceIDsForAccountResourceTypeFromES(s.es, res.DescribeJob.SourceID, res.DescribeJob.ResourceType, searchAfter, 1000)
		if err != nil {
			return err
		}

		if len(esResp.Hits.Hits) == 0 {
			break
		}

		for _, hit := range esResp.Hits.Hits {
			esResourceIDs = append(esResourceIDs, hit.Source.ResourceID)
			searchAfter = hit.Sort
		}
	}

	for _, esResourceID := range esResourceIDs {
		exists := false
		for _, describedResourceID := range res.DescribedResourceIDs {
			if esResourceID == describedResourceID {
				exists = true
				break
			}
		}

		if !exists {
			fmt.Println("deleting ", esResourceID)
			resource := es.Resource{
				ID: esResourceID,
			}
			keys, idx := resource.KeysAndIndex()
			key := kafka.HashOf(keys...)
			kafkaMsgs = append(kafkaMsgs, &sarama.ProducerMessage{
				Topic: s.kafkaResourcesTopic,
				Key:   sarama.StringEncoder(key),
				Headers: []sarama.RecordHeader{
					{
						Key:   []byte(kafka.EsIndexHeader),
						Value: []byte(idx),
					},
				},
				Value: nil,
			})

			lookupResource := es.LookupResource{
				ResourceID:   esResourceID,
				ResourceType: res.DescribeJob.ResourceType,
				SourceType:   source.Type(res.DescribeJob.SourceType),
			}
			keys, idx = lookupResource.KeysAndIndex()
			key = kafka.HashOf(keys...)
			kafkaMsgs = append(kafkaMsgs, &sarama.ProducerMessage{
				Topic: s.kafkaResourcesTopic,
				Key:   sarama.StringEncoder(key),
				Headers: []sarama.RecordHeader{
					{
						Key:   []byte(kafka.EsIndexHeader),
						Value: []byte(idx),
					},
				},
				Value: nil,
			})
		}
	}

	if err := s.kafkaProducer.SendMessages(kafkaMsgs); err != nil {
		if errs, ok := err.(sarama.ProducerErrors); ok {
			for _, e := range errs {
				s.logger.Error("Falied calling SendMessages", zap.Error(fmt.Errorf("Failed to persist resource[%s] in kafka topic[%s]: %s\nMessage: %v\n", e.Msg.Key, e.Msg.Topic, e.Error(), e.Msg)))
			}
		}

		return err
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
		for _, v := range results {
			if _, ok := jobIDToStatus[v.DescribeSourceJobID]; !ok {
				jobIDToStatus[v.DescribeSourceJobID] = map[api.DescribeResourceJobStatus]int{
					api.DescribeResourceJobCreated:      0,
					api.DescribeResourceJobQueued:       0,
					api.DescribeResourceJobCloudTimeout: 0,
					api.DescribeResourceJobFailed:       0,
					api.DescribeResourceJobSucceeded:    0,
				}
			}

			jobIDToStatus[v.DescribeSourceJobID][v.DescribeResourceJobStatus] = v.DescribeResourceJobCount
		}

		for id, status := range jobIDToStatus {
			// If any CREATED or QUEUED, job is still in progress
			if status[api.DescribeResourceJobCreated] > 0 ||
				status[api.DescribeResourceJobQueued] > 0 {
				continue
			}

			// If any FAILURE, job is completed with failure
			if status[api.DescribeResourceJobFailed] > 0 || status[api.DescribeResourceJobCloudTimeout] > 0 {
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
