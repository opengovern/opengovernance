package describe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ProtonMail/gopenpgp/v2/helper"
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
			)
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

func (s *Scheduler) RunDescribeConnectionJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the JobResults queue")

	msgs, err := s.describeConnectionJobResultQueue.Consume()
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

			var result DescribeConnectionJobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				s.logger.Error("Failed to unmarshal DescribeConnectionJobResult results", zap.Error(err))
				err = msg.Nack(false, false)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			requeue, err := s.consumeDescribeConnectionJobResults(result)
			if err != nil {
				s.logger.Error("Failed to consumeDescribeConnectionJobResults", zap.Error(err))
				err = msg.Nack(false, requeue)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			if err := msg.Ack(false); err != nil {
				s.logger.Error("Failed acking message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateDescribeResourceJobsTimedOut(s.describeTimeoutHours)
			if err != nil {
				s.logger.Error("Failed to update timed out DescribeResourceJobs", zap.Error(err))
			}
		}
	}
}

func (s *Scheduler) consumeDescribeConnectionJobResults(result DescribeConnectionJobResult) (requeue bool, err error) {
	var kafkaMsgs []sarama.ProducerMessage
	for jobID, res := range result.Result {
		s.logger.Info("Processing JobResult for Job",
			zap.Uint("jobId", jobID),
			zap.String("status", string(res.Status)),
		)

		if strings.Contains(res.Error, "ThrottlingException") ||
			strings.Contains(res.Error, "Rate exceeded") ||
			strings.Contains(res.Error, "RateExceeded") ||
			res.Status == api.DescribeResourceJobCloudTimeout {

			// sent it to describe jobs
			s.logger.Info("Needs to be retried",
				zap.Uint("jobId", jobID),
				zap.String("status", string(res.Status)),
			)
			res.DescribeJob.RetryCounter++
			if res.DescribeJob.RetryCounter > 5 {
				res.Status = api.DescribeResourceJobFailed
				res.Error = fmt.Sprintf("Retries exhuasted - original error: %s", res.Error)
			} else {
				err := s.describeJobQueue.Publish(res.DescribeJob)
				if err != nil {
					return true, err
				}
				continue
			}
		}

		esResp, err := es.GetResourceIDsForAccountResourceTypeFromES(s.es, res.DescribeJob.SourceID, res.DescribeJob.ResourceType)
		if err != nil {
			return true, err
		}
		var esResourceIDs []string
		for _, bucket := range esResp.Aggregations.ResourceIDFilter.Buckets {
			esResourceIDs = append(esResourceIDs, bucket.Key)
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
				kafkaMsgs = append(kafkaMsgs, sarama.ProducerMessage{
					Topic: s.kafkaResourcesTopic,
					Key:   sarama.StringEncoder(esResourceID),
					Headers: []sarama.RecordHeader{
						{
							Key:   []byte(kafka.EsIndexHeader),
							Value: []byte(ResourceTypeToESIndex(res.DescribeJob.ResourceType)),
						},
					},
					Value: nil,
				})
			}
		}

		err = s.db.UpdateDescribeResourceJobStatus(res.JobID, res.Status, res.Error)
		if err != nil {
			return true, err
		}
	}
	return false, nil
}

func (s *Scheduler) RunCloudNativeDescribeConnectionJobResourcesConsumer() {
	t := time.NewTicker(JobCompletionInterval)
	defer t.Stop()

	for ; ; <-t.C {
		s.cloudNativeDescribeConnectionJobResourcesConsume()
	}
}

type cloudNativeDescribeConnectionJobResourceOutput struct {
	ID      int    `json:"id"`
	Payload string `json:"payload"`
}

func (s *Scheduler) cloudNativeDescribeConnectionJobResourcesConsume() {
	for {
		s.logger.Info("Checking for cloud native describe connection job resources")
		httpClient := &http.Client{
			Timeout: 1 * time.Minute,
		}
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/OutputSQLReader", s.cloudNativeAPIBaseURL), nil)
		if err != nil {
			s.logger.Error("Failed to create http request", zap.Error(err))
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("x-kaytu-cloud-auth-key", s.cloudNativeAPIAuthKey)
		resp, err := httpClient.Do(req)
		if err != nil {
			s.logger.Error("Failed to send OutputSQLReader http request", zap.Error(err))
			return
		}
		defer resp.Body.Close()
		resBody, err := io.ReadAll(resp.Body)
		if err != nil {
			s.logger.Error("Failed to read OutputSQLReader response body", zap.Error(err))
			return
		}
		if resp.StatusCode != http.StatusOK {
			s.logger.Error("Http request OutputSQLReader status not ok", zap.Int("status", resp.StatusCode), zap.String("body", string(resBody)))
			return
		}

		res := make([]cloudNativeDescribeConnectionJobResourceOutput, 0)
		if err := json.Unmarshal(resBody, &res); err != nil {
			s.logger.Error("Failed to unmarshal OutputSQLReader response body", zap.Error(err))
			return
		}
		if len(res) == 0 {
			return
		}

		successfulIds, err := s.processCloudNativeDescribeConnectionJobResourcesEvents(res)
		if err != nil {
			s.logger.Error("Failed to process events", zap.Error(err))
			return
		}
		jsonIds, err := json.Marshal(successfulIds)
		if err != nil {
			s.logger.Error("Failed to marshal successful ids", zap.Error(err))
			return
		}

		req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("%s/OutputSQLAck", s.cloudNativeAPIBaseURL), bytes.NewBuffer(jsonIds))
		if err != nil {
			s.logger.Error("Failed to create http request", zap.Error(err))
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("x-kaytu-cloud-auth-key", s.cloudNativeAPIAuthKey)
		resp, err = httpClient.Do(req)
		if err != nil {
			s.logger.Error("Failed to send OutputSQLAck http request", zap.Error(err))
			return
		}
		defer resp.Body.Close()
		resBody, err = io.ReadAll(resp.Body)
		if err != nil {
			s.logger.Error("Failed to read OutputSQLAck response body", zap.Error(err))
			return
		}
		if resp.StatusCode != http.StatusOK {
			s.logger.Error("Http request OutputSQLAck status not ok", zap.Int("status", resp.StatusCode), zap.String("body", string(resBody)))
			return
		}
	}
}

func (s *Scheduler) processCloudNativeDescribeConnectionJobResourcesEvents(events []cloudNativeDescribeConnectionJobResourceOutput) ([]int, error) {
	s.logger.Info("Received events from cloud native describe connection job resources sql", zap.Int("eventCount", len(events)))
	successfulIDs := make([]int, 0)
	for _, event := range events {
		var connectionWorkerResourcesResult CloudNativeConnectionWorkerResult
		err := json.Unmarshal([]byte(event.Payload), &connectionWorkerResourcesResult)
		if err != nil {
			s.logger.Error("Error unmarshalling event", zap.Error(err))
			continue
		}

		job, err := s.db.GetCloudNativeDescribeSourceJob(connectionWorkerResourcesResult.JobID)
		if err != nil {
			s.logger.Error("Error getting cloud native describe source job", zap.Error(err))
			continue
		}
		if job == nil {
			successfulIDs = append(successfulIDs, event.ID)
			continue
		}

		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/OutputReadBlob?containerName=%s&blobName=%s", s.cloudNativeAPIBaseURL, connectionWorkerResourcesResult.ContainerName, connectionWorkerResourcesResult.BlobName), nil)
		if err != nil {
			s.logger.Error("Failed to create OutputReadBlob http request", zap.Error(err))
			continue
		}
		httpClient := &http.Client{
			Timeout: 1 * time.Minute,
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("x-kaytu-cloud-auth-key", s.cloudNativeAPIAuthKey)
		resp, err := httpClient.Do(req)
		if err != nil {
			s.logger.Error("Failed to get blob stream", zap.Error(err))
			continue
		}
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			s.logger.Error("Failed to read OutputReadBlob http response", zap.Error(err))
			continue
		}
		resp.Body.Close()

		var cloudNativeConnectionWorkerData []CloudNativeConnectionWorkerData
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			s.logger.Error("Failed to get blob stream", zap.Error(err), zap.String("response", string(respBody)))
			continue
		} else if resp.StatusCode == http.StatusOK {
			var data api.GetDataResponse
			err = json.Unmarshal(respBody, &data)
			if err != nil {
				s.logger.Error("Failed to unmarshal OutputReadBlob response", zap.Error(err))
				continue
			}
			var encryptedCNCData []string
			err = json.Unmarshal([]byte(data.Data), &encryptedCNCData)
			if err != nil {
				s.logger.Error("Failed to unmarshal data into messages", zap.Error(err))
				continue
			}
			for _, encryptedData := range encryptedCNCData {
				decrypted, err := helper.DecryptMessageArmored(job.ResultEncryptionPrivateKey, nil, encryptedData)
				if err != nil {
					s.logger.Error("Failed to decrypt blob", zap.Error(err))
					continue
				}
				var cncData CloudNativeConnectionWorkerData
				err = json.Unmarshal([]byte(decrypted), &cncData)
				if err != nil {
					s.logger.Error("Failed to unmarshal data into messages", zap.Error(err))
					continue
				}
				cloudNativeConnectionWorkerData = append(cloudNativeConnectionWorkerData, cncData)
			}
		}
		producer, err := sarama.NewSyncProducerFromClient(s.kafkaClient)
		for _, cncData := range cloudNativeConnectionWorkerData {
			saramaMessages := make([]*sarama.ProducerMessage, 0, len(cncData.JobData))
			for _, message := range cncData.JobData {
				saramaMessage := sarama.ProducerMessage{
					Topic:   message.Topic,
					Key:     message.Key,
					Value:   message.Value,
					Headers: message.Headers,
				}
				//isIdentical, err := s.compareMessageToCurrentState(saramaMessage)
				//if err != nil {
				//	s.logger.Error("Failed to compare message to current state", zap.Error(err))
				//} else {
				//	if isIdentical {
				//		s.logger.Info("New message is identical to current state, overwriting to update timestamp")
				//	} else {
				//		s.logger.Info("New message is not identical to current state, overwriting")
				//	}
				//}
				saramaMessages = append(saramaMessages, &saramaMessage)
			}

			if err != nil {
				s.logger.Error("Failed to create producer", zap.Error(err))
				continue
			}
			if err := producer.SendMessages(saramaMessages); err != nil {
				if errs, ok := err.(sarama.ProducerErrors); ok {
					for _, e := range errs {
						s.logger.Error("Failed calling SendMessages", zap.Error(fmt.Errorf("Failed to persist resource[%s] in kafka topic[%s]: %s, message size: %d\n", e.Msg.Key, e.Msg.Topic, e.Error(), e.Msg.Value.Length())))
					}
				}
				continue
			}

			if len(saramaMessages) != 0 {
				s.logger.Info("Successfully sent messages to kafka", zap.Int("count", len(saramaMessages)))
			}

			err = s.describeConnectionJobResultQueue.Publish(cncData.JobResult)
			if err != nil {
				s.logger.Error("Failed calling describeConnectionJobResultQueue.Publish", zap.Error(err))
				continue
			}
		}
		if err := producer.Close(); err != nil {
			s.logger.Error("Failed to close producer", zap.Error(err))
			continue
		}
		successfulIDs = append(successfulIDs, event.ID)
	}

	s.logger.Info("Processed events from cloud native describe connection job resources sql", zap.Int("eventCount", len(successfulIDs)))
	return successfulIDs, nil
}

func (s *Scheduler) compareMessageToCurrentState(message sarama.ProducerMessage) (bool, error) {
	index := ""
	for _, header := range message.Headers {
		if string(header.Key) == kafka.EsIndexHeader {
			index = string(header.Value)
		}
	}
	if index == "" {
		return false, fmt.Errorf("no index header found to compare")
	}

	id := string(message.Key.(sarama.StringEncoder))

	switch index {
	case InventorySummaryIndex:
		currentResource, err := es.FetchLookupResourceByID(s.es, index, id)
		if err != nil {
			s.logger.Error("failed to fetch current resource", zap.String("error", err.Error()), zap.String("index", index), zap.String("id", id))
			return false, err
		}
		if currentResource == nil {
			return false, nil
		}
		newResource := es.LookupResource{}
		err = json.Unmarshal(message.Value.(sarama.ByteEncoder), &newResource)

		currentResourceBytes, err := json.Marshal(currentResource)
		if err != nil {
			s.logger.Error("failed to marshal current resource", zap.String("error", err.Error()), zap.String("index", index), zap.String("id", id))
			return false, err
		}
		newResourceBytes, err := json.Marshal(newResource)
		if err != nil {
			s.logger.Error("failed to marshal new resource", zap.String("error", err.Error()), zap.String("index", index), zap.String("id", id))
			return false, err
		}

		if string(currentResourceBytes) == string(newResourceBytes) {
			return true, nil
		}
		return false, nil
	default:
		currentResource, err := es.FetchResourceByID(s.es, index, id)
		if err != nil {
			s.logger.Error("failed to fetch current resource", zap.String("error", err.Error()), zap.String("index", index), zap.String("id", id))
			return false, err
		}
		if currentResource == nil {
			return false, nil
		}
		newResource := es.Resource{}
		err = json.Unmarshal(message.Value.(sarama.ByteEncoder), &newResource)

		currentResourceBytes, err := json.Marshal(currentResource.Description)
		if err != nil {
			s.logger.Error("failed to marshal current resource", zap.String("error", err.Error()), zap.String("index", index), zap.String("id", id))
			return false, err
		}
		newResourceBytes, err := json.Marshal(newResource.Description)
		if err != nil {
			s.logger.Error("failed to marshal new resource", zap.String("error", err.Error()), zap.String("index", index), zap.String("id", id))
			return false, err
		}

		if string(currentResourceBytes) == string(newResourceBytes) {
			return true, nil
		}
		return false, nil
	}
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
				status[api.DescribeResourceJobQueued] > 0 ||
				status[api.DescribeResourceJobCloudTimeout] > 0 {
				continue
			}

			// If any FAILURE, job is completed with failure
			if status[api.DescribeResourceJobFailed] > 0 {
				err := s.db.UpdateDescribeSourceJob(id, api.DescribeSourceJobCompletedWithFailure)
				if err != nil {
					s.logger.Error("Failed to update DescribeSourceJob status\n",
						zap.Uint("jobId", id),
						zap.String("status", string(api.DescribeSourceJobCompletedWithFailure)),
						zap.Error(err),
					)
				}

				job, err := s.db.GetDescribeSourceJob(id)
				if err != nil {
					s.logger.Error("Failed to call summarizer\n",
						zap.Uint("jobId", id),
						zap.Error(err),
					)
				} else if job == nil {
					s.logger.Error("Failed to find the job for summarizer\n",
						zap.Uint("jobId", id),
						zap.Error(err),
					)
				} else {
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
