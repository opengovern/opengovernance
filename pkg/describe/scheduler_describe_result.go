package describe

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	"strings"
	"time"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"

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
			if result.Status == api.DescribeResourceJobFailed {
				ResourcesDescribedCount.WithLabelValues(strings.ToLower(result.DescribeJob.SourceType.String()), "failure").Inc()
			}
			if err := s.db.UpdateDescribeConnectionJobStatus(result.JobID, result.Status, errStr, errCodeStr, int64(len(result.DescribedResourceIDs))); err != nil {
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
			awsResources := aws.ListResourceTypes()
			for _, r := range awsResources {
				var interval int64
				resourceType, err := aws.GetResourceType(r)
				if err != nil {
					s.logger.Error(fmt.Sprintf("failed to get resource type %s", r), zap.Error(err))
				}
				if resourceType.FastDiscovery {
					interval = s.describeIntervalHours
				} else if resourceType.CostDiscovery {
					interval = 24
				} else {
					interval = s.fullDiscoveryIntervalHours
				}
				err = s.db.UpdateResourceTypeDescribeConnectionJobsTimedOut(r, interval)
				//s.logger.Warn(fmt.Sprintf("describe resource job timed out on %s:", r), zap.Error(err))
				//DescribeResourceJobsCount.WithLabelValues("failure", "timedout_aws").Inc()
				ResourcesDescribedCount.WithLabelValues("aws", "failure").Inc()
				if err != nil {
					s.logger.Error(fmt.Sprintf("failed to update timed out DescribeResourceJobs on %s:", r), zap.Error(err))
				}
			}
			azureResources := azure.ListResourceTypes()
			for _, r := range azureResources {
				var interval int64
				resourceType, err := azure.GetResourceType(r)
				if err != nil {
					s.logger.Error(fmt.Sprintf("failed to get resource type %s", r), zap.Error(err))
				}
				if resourceType.FastDiscovery {
					interval = s.describeIntervalHours
				} else if resourceType.CostDiscovery {
					interval = 24
				} else {
					interval = s.fullDiscoveryIntervalHours
				}
				err = s.db.UpdateResourceTypeDescribeConnectionJobsTimedOut(r, interval)
				//s.logger.Warn(fmt.Sprintf("describe resource job timed out on %s:", r), zap.Error(err))
				//DescribeResourceJobsCount.WithLabelValues("failure", "timedout_azure").Inc()
				ResourcesDescribedCount.WithLabelValues("azure", "failure").Inc()
				if err != nil {
					s.logger.Error(fmt.Sprintf("failed to update timed out DescribeResourceJobs on %s:", r), zap.Error(err))
				}
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
			CleanupJobCount.WithLabelValues("failure").Inc()
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
					SourceID:     res.DescribeJob.SourceID,
					ResourceType: res.DescribeJob.ResourceType,
					SourceType:   res.DescribeJob.SourceType,
				}
				keys, idx := resource.KeysAndIndex()
				msg := kafka.Msg(kafka.HashOf(keys...), nil, idx, s.kafkaResourcesTopic, confluent_kafka.PartitionAny)
				msgs = append(msgs, msg)

				lookupResource := es.LookupResource{
					ResourceID:   esResourceID,
					SourceID:     res.DescribeJob.SourceID,
					ResourceType: res.DescribeJob.ResourceType,
					SourceType:   res.DescribeJob.SourceType,
				}
				lookUpKeys, lookUpIdx := lookupResource.KeysAndIndex()
				msg = kafka.Msg(kafka.HashOf(lookUpKeys...), nil, lookUpIdx, s.kafkaResourcesTopic, confluent_kafka.PartitionAny)
				msgs = append(msgs, msg)
				if err != nil {
					CleanupJobCount.WithLabelValues("failure").Inc()
					return err
				}
			}
		}

		i := 0
		for {
			_, err = kafka.SyncSend(s.logger, s.kafkaProducer, msgs, nil)
			if err != nil {
				s.logger.Error("failed to send delete message to kafka",
					zap.Uint("jobId", res.JobID),
					zap.String("connection_id", res.DescribeJob.SourceID),
					zap.String("resource_type", res.DescribeJob.ResourceType),
					zap.Error(err))
				if i > 10 {
					CleanupJobCount.WithLabelValues("failure").Inc()
					return err
				}
				i++
				continue
			}
			break
		}
		deletedCount += len(msgs)
	}
	s.logger.Info("deleted old resources",
		zap.Uint("jobId", res.JobID),
		zap.String("connection_id", res.DescribeJob.SourceID),
		zap.String("resource_type", res.DescribeJob.ResourceType),
		zap.Int("deleted_count", deletedCount))

	CleanupJobCount.WithLabelValues("successful").Inc()
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
		_, err = kafka.SyncSend(s.logger, s.kafkaProducer, msgs, nil)
		if err != nil {
			s.logger.Error("failed to send delete message to kafka", zap.Error(err))
			return err
		}
	}

	return nil
}
