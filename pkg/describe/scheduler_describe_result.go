package describe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	authApi "github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	es2 "github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/ticker"
	"github.com/kaytu-io/open-governance/pkg/describe/api"
	"github.com/kaytu-io/open-governance/pkg/describe/es"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

func (s *Scheduler) UpdateDescribedResourceCountScheduler() error {
	s.logger.Info("DescribedResourceCount update scheduler started")

	t := ticker.NewTicker(1*time.Minute, time.Second*10)
	defer t.Stop()

	for ; ; <-t.C {
		s.UpdateDescribedResourceCount()
	}
}

func (s *Scheduler) UpdateDescribedResourceCount() {
	s.logger.Info("Updating DescribedResourceCount")
	AwsFailedCount, err := s.db.CountJobsWithStatus(8, source.CloudAWS, api.DescribeResourceJobFailed)
	if err != nil {
		s.logger.Error("Failed to count described resources",
			zap.String("connector", "AWS"),
			zap.String("status", "failed"),
			zap.Error(err))
		return
	}
	ResourcesDescribedCount.WithLabelValues("aws", "failure").Set(float64(*AwsFailedCount))
	AzureFailedCount, err := s.db.CountJobsWithStatus(8, source.CloudAzure, api.DescribeResourceJobFailed)
	if err != nil {
		s.logger.Error("Failed to count described resources",
			zap.String("connector", "Azure"),
			zap.String("status", "failed"),
			zap.Error(err))
		return
	}
	ResourcesDescribedCount.WithLabelValues("azure", "failure").Set(float64(*AzureFailedCount))
	AwsSucceededCount, err := s.db.CountJobsWithStatus(8, source.CloudAWS, api.DescribeResourceJobSucceeded)
	if err != nil {
		s.logger.Error("Failed to count described resources",
			zap.String("connector", "AWS"),
			zap.String("status", "successful"),
			zap.Error(err))
		return
	}
	ResourcesDescribedCount.WithLabelValues("aws", "successful").Set(float64(*AwsSucceededCount))
	AzureSucceededCount, err := s.db.CountJobsWithStatus(8, source.CloudAzure, api.DescribeResourceJobSucceeded)
	if err != nil {
		s.logger.Error("Failed to count described resources",
			zap.String("connector", "Azure"),
			zap.String("status", "successful"),
			zap.Error(err))
		return
	}
	ResourcesDescribedCount.WithLabelValues("azure", "successful").Set(float64(*AzureSucceededCount))
}

func (s *Scheduler) RunDescribeJobResultsConsumer(ctx context.Context) error {
	s.logger.Info("Consuming messages from the JobResults queue")

	consumeCtx, err := s.jq.Consume(
		ctx,
		"describe-receiver",
		DescribeStreamName,
		[]string{DescribeResultsQueueName},
		"describe-receiver",
		func(msg jetstream.Msg) {
			var result DescribeJobResult
			if err := json.Unmarshal(msg.Data(), &result); err != nil {
				ResultsProcessedCount.WithLabelValues("", "failure").Inc()

				s.logger.Error("failed to consume message from describeJobResult", zap.Error(err))

				// the job cannot be parsed into json, so send ack and throw message away.
				if err := msg.Ack(); err != nil {
					s.logger.Error("failure while sending ack for message", zap.Error(err))
				}

				return
			}

			s.logger.Info("Processing JobResult for Job",
				zap.Uint("jobId", result.JobID),
				zap.String("status", string(result.Status)),
			)

			var deletedCount int64
			if s.DoDeleteOldResources && result.Status == api.DescribeResourceJobSucceeded {
				result.Status = api.DescribeResourceJobOldResourceDeletion

				dlc, err := s.cleanupOldResources(ctx, result)
				if err != nil {
					ResultsProcessedCount.WithLabelValues(string(result.DescribeJob.SourceType), "failure").Inc()
					s.logger.Error("failed to cleanupOldResources", zap.Error(err))

					if err := msg.Nak(); err != nil {
						s.logger.Error("failure while sending not-ack for message", zap.Error(err))
					}

					return
				}

				deletedCount = dlc
				if deletedCount == 0 {
					result.Status = api.DescribeResourceJobSucceeded
				}
			}

			errStr := strings.ReplaceAll(result.Error, "\x00", "")
			errCodeStr := strings.ReplaceAll(result.ErrorCode, "\x00", "")
			if errCodeStr == "" {
				if strings.Contains(errStr, "exceeded maximum number of attempts") {
					errCodeStr = "TooManyRequestsException"
				} else if strings.Contains(errStr, "context deadline exceeded") {
					errCodeStr = "ContextDeadlineExceeded"
				}
			}

			s.logger.Info("updating job status", zap.Uint("jobID", result.JobID), zap.String("status", string(result.Status)))
			if err := s.db.UpdateDescribeConnectionJobStatus(result.JobID, result.Status, errStr, errCodeStr, int64(len(result.DescribedResourceIDs)), deletedCount); err != nil {
				ResultsProcessedCount.WithLabelValues(string(result.DescribeJob.SourceType), "failure").Inc()

				s.logger.Error("failed to UpdateDescribeResourceJobStatus", zap.Error(err))

				if err := msg.Nak(); err != nil {
					s.logger.Error("failure while sending not-ack for message", zap.Error(err))
				}

				return
			}

			ResultsProcessedCount.WithLabelValues(string(result.DescribeJob.SourceType), "successful").Inc()

			if err := msg.Ack(); err != nil {
				s.logger.Error("failure while sending ack for message", zap.Error(err))
			}
		},
	)
	if err != nil {
		return err
	}

	t := ticker.NewTicker(JobTimeoutCheckInterval, time.Minute*1)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			s.handleTimeoutForDiscoveryJobs()
		case <-ctx.Done():
			consumeCtx.Drain()
			consumeCtx.Stop()

			return nil
		}
	}
}

func (s *Scheduler) handleTimeoutForDiscoveryJobs() {
	awsResources := aws.ListResourceTypes()
	for _, r := range awsResources {
		var interval time.Duration
		resourceType, err := aws.GetResourceType(r)
		if err != nil {
			s.logger.Error(fmt.Sprintf("failed to get resource type %s", r), zap.Error(err))
		}
		if resourceType.FastDiscovery {
			interval = s.describeIntervalHours
		} else if resourceType.CostDiscovery {
			interval = s.costDiscoveryIntervalHours
		} else {
			interval = s.fullDiscoveryIntervalHours
		}

		if _, err := s.db.UpdateResourceTypeDescribeConnectionJobsTimedOut(r, interval); err != nil {
			s.logger.Error(fmt.Sprintf("failed to update timed out DescribeResourceJobs on %s:", r), zap.Error(err))
		}
	}
	azureResources := azure.ListResourceTypes()
	for _, r := range azureResources {
		var interval time.Duration
		resourceType, err := azure.GetResourceType(r)
		if err != nil {
			s.logger.Error(fmt.Sprintf("failed to get resource type %s", r), zap.Error(err))
		}
		if resourceType.FastDiscovery {
			interval = s.describeIntervalHours
		} else if resourceType.CostDiscovery {
			interval = s.costDiscoveryIntervalHours
		} else {
			interval = s.fullDiscoveryIntervalHours
		}

		if _, err := s.db.UpdateResourceTypeDescribeConnectionJobsTimedOut(r, interval); err != nil {
			s.logger.Error(fmt.Sprintf("failed to update timed out DescribeResourceJobs on %s:", r), zap.Error(err))
		}
	}
}

func (s *Scheduler) cleanupOldResources(ctx context.Context, res DescribeJobResult) (int64, error) {
	var searchAfter []any

	isCostResourceType := false
	if strings.ToLower(res.DescribeJob.ResourceType) == "microsoft.costmanagement/costbyresourcetype" ||
		strings.ToLower(res.DescribeJob.ResourceType) == "aws::costexplorer::byservicedaily" {
		isCostResourceType = true
	}

	var additionalFilters []map[string]any
	if isCostResourceType {
		additionalFilters = append(additionalFilters, map[string]any{
			"range": map[string]any{"cost_date": map[string]any{"lt": time.Now().AddDate(0, -2, -1).UnixMilli()}},
		})
	}

	deletedCount := 0

	s.logger.Info("starting to schedule deleting old resources",
		zap.Uint("jobId", res.JobID),
		zap.String("connection_id", res.DescribeJob.SourceID),
		zap.String("resource_type", res.DescribeJob.ResourceType),
	)

	for {
		esResp, err := es.GetResourceIDsForAccountResourceTypeFromES(
			ctx,
			s.es,
			res.DescribeJob.SourceID,
			res.DescribeJob.ResourceType,
			additionalFilters,
			searchAfter,
			1000)
		if err != nil {
			CleanupJobCount.WithLabelValues("failure").Inc()
			s.logger.Error("CleanJob failed",
				zap.Error(err))
			return 0, err
		}

		if len(esResp.Hits.Hits) == 0 {
			break
		}
		task := es.DeleteTask{
			DiscoveryJobID: res.JobID,
			ConnectionID:   res.DescribeJob.SourceID,
			ResourceType:   res.DescribeJob.ResourceType,
			Connector:      res.DescribeJob.SourceType,
			TaskType:       es.DeleteTaskTypeResource,
		}

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

			if !exists || isCostResourceType {
				OldResourcesDeletedCount.WithLabelValues(string(res.DescribeJob.SourceType)).Inc()
				resource := es2.Resource{
					ID:           esResourceID,
					SourceID:     res.DescribeJob.SourceID,
					ResourceType: res.DescribeJob.ResourceType,
					SourceType:   res.DescribeJob.SourceType,
				}
				keys, idx := resource.KeysAndIndex()
				deletedCount += 1
				task.DeletingResources = append(task.DeletingResources, es.DeletingResource{
					Key:        []byte(es2.HashOf(keys...)),
					ResourceID: esResourceID,
					Index:      idx,
				})

				lookupResource := es2.LookupResource{
					ResourceID:   esResourceID,
					SourceID:     res.DescribeJob.SourceID,
					ResourceType: res.DescribeJob.ResourceType,
					SourceType:   res.DescribeJob.SourceType,
				}
				lookUpKeys, lookUpIdx := lookupResource.KeysAndIndex()
				deletedCount += 1
				task.DeletingResources = append(task.DeletingResources, es.DeletingResource{
					Key:        []byte(es2.HashOf(lookUpKeys...)),
					ResourceID: esResourceID,
					Index:      lookUpIdx,
				})

				if err != nil {
					CleanupJobCount.WithLabelValues("failure").Inc()
					s.logger.Error("CleanJob failed",
						zap.Error(err))
					return 0, err
				}
			}
		}

		i := 0
		for {
			taskKeys, taskIdx := task.KeysAndIndex()
			task.EsID = es2.HashOf(taskKeys...)
			task.EsIndex = taskIdx

			if len(task.DeletingResources) > 0 {
				if _, err := s.sinkClient.Ingest(&httpclient.Context{UserRole: authApi.InternalRole}, []es2.Doc{task}); err != nil {
					s.logger.Error("failed to send delete message to elastic",
						zap.Uint("jobId", res.JobID),
						zap.String("connection_id", res.DescribeJob.SourceID),
						zap.String("resource_type", res.DescribeJob.ResourceType),
						zap.Error(err))
					if i > 10 {
						CleanupJobCount.WithLabelValues("failure").Inc()
						return 0, err
					}
					i++
					continue
				}
			}
			break
		}
	}

	s.logger.Info("scheduled deleting old resources",
		zap.Uint("jobId", res.JobID),
		zap.String("connection_id", res.DescribeJob.SourceID),
		zap.String("resource_type", res.DescribeJob.ResourceType),
		zap.Int("deleted_count", deletedCount))

	CleanupJobCount.WithLabelValues("successful").Inc()
	return int64(deletedCount), nil
}

func (s *Scheduler) cleanupDescribeResourcesForConnections(ctx context.Context, connectionIds []string) {
	for _, connectionId := range connectionIds {
		var searchAfter []any
		for {
			esResp, err := es.GetResourceIDsForAccountFromES(ctx, s.es, connectionId, searchAfter, 1000)
			if err != nil {
				s.logger.Error("failed to get resource ids from es", zap.Error(err))
				break
			}

			if len(esResp.Hits.Hits) == 0 {
				break
			}
			deletedCount := 0
			for _, hit := range esResp.Hits.Hits {
				searchAfter = hit.Sort

				resource := es2.Resource{
					ID:           hit.Source.ResourceID,
					SourceID:     hit.Source.SourceID,
					ResourceType: strings.ToLower(hit.Source.ResourceType),
					SourceType:   hit.Source.SourceType,
				}
				keys, idx := resource.KeysAndIndex()
				deletedCount += 1
				key := es2.HashOf(keys...)
				resource.EsID = key
				resource.EsIndex = idx
				err = s.es.Delete(key, idx)
				if err != nil {
					s.logger.Error("failed to delete resource from open-search", zap.Error(err))
					return
				}

				lookupResource := es2.LookupResource{
					ResourceID:   hit.Source.ResourceID,
					SourceID:     hit.Source.SourceID,
					ResourceType: strings.ToLower(hit.Source.ResourceType),
					SourceType:   hit.Source.SourceType,
				}
				deletedCount += 1
				keys, idx = lookupResource.KeysAndIndex()
				key = es2.HashOf(keys...)
				lookupResource.EsID = key
				lookupResource.EsIndex = idx
				err = s.es.Delete(key, idx)
				if err != nil {
					s.logger.Error("failed to delete lookup from open-search", zap.Error(err))
					return
				}
			}

			s.logger.Info("deleted old resources", zap.Int("deleted_count", deletedCount), zap.String("connection_id", connectionId))
		}
	}

	return
}

func (s *Scheduler) cleanupDescribeResourcesForConnectionAndResourceType(connectionId, resourceType string) error {
	root := make(map[string]any)
	root["query"] = map[string]any{
		"bool": map[string]any{
			"filter": []any{
				map[string]any{
					"term": map[string]any{
						"source_id": connectionId,
					},
				},
				map[string]any{
					"term": map[string]any{
						"resource_type": strings.ToLower(resourceType),
					},
				},
			},
		},
	}
	query, err := json.Marshal(root)
	if err != nil {
		return err
	}

	index := es2.ResourceTypeToESIndex(resourceType)
	res, err := s.es.ES().DeleteByQuery([]string{index}, bytes.NewReader(query))
	if err != nil {
		return err
	}

	kaytu.CloseSafe(res)

	res, err = s.es.ES().DeleteByQuery([]string{InventorySummaryIndex}, bytes.NewReader(query))
	if err != nil {
		return err
	}

	kaytu.CloseSafe(res)
	return nil
}
