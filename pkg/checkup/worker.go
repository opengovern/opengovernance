package checkup

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/queue"

	"github.com/kaytu-io/kaytu-engine/pkg/onboard/client"

	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/zap"
)

type Worker struct {
	id             string
	jobQueue       queue.Interface
	jobResultQueue queue.Interface
	logger         *zap.Logger
	pusher         *push.Pusher
	onboardClient  client.OnboardServiceClient
}

func InitializeWorker(
	id string,
	rabbitMQUsername string,
	rabbitMQPassword string,
	rabbitMQHost string,
	rabbitMQPort int,
	checkupJobQueue string,
	checkupJobResultQueue string,
	logger *zap.Logger,
	prometheusPushAddress string,
	onboardBaseURL string,
) (w *Worker, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	}

	w = &Worker{id: id}
	defer func() {
		if err != nil && w != nil {
			w.Stop()
		}
	}()

	qCfg := queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = checkupJobQueue
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = w.id
	checkupQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobQueue = checkupQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = checkupJobResultQueue
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = w.id
	checkupResultsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobResultQueue = checkupResultsQueue

	w.logger = logger

	w.pusher = push.New(prometheusPushAddress, "checkup-worker")
	w.pusher.Collector(DoCheckupJobsCount).
		Collector(DoCheckupJobsDuration)

	w.onboardClient = client.NewOnboardServiceClient(onboardBaseURL, nil)
	return w, nil
}

func (w *Worker) Run() error {
	msgs, err := w.jobQueue.Consume()
	if err != nil {
		return err
	}

	w.logger.Error("Waiting indefinitly for messages. To exit press CTRL+C")
	for msg := range msgs {
		var job Job
		if err := json.Unmarshal(msg.Body, &job); err != nil {
			w.logger.Error("Failed to unmarshal task", zap.Error(err))
			err = msg.Nack(false, false)
			if err != nil {
				w.logger.Error("Failed nacking message", zap.Error(err))
			}
			continue
		}
		w.logger.Info("Processing job", zap.Int("jobID", int(job.JobID)))
		result := job.Do(w.onboardClient, w.logger)
		w.logger.Info("Publishing job result", zap.Int("jobID", int(job.JobID)))
		err := w.jobResultQueue.Publish(result)
		if err != nil {
			w.logger.Error("Failed to send results to queue: %s", zap.Error(err))
		}

		if err := msg.Ack(false); err != nil {
			w.logger.Error("Failed acking message", zap.Error(err))
		}

		err = w.pusher.Push()
		if err != nil {
			w.logger.Error("Failed to push metrics", zap.Error(err))
		}
	}

	return fmt.Errorf("checkup jobs channel is closed")
}

func (w *Worker) Stop() {
	w.pusher.Push()

	if w.jobQueue != nil {
		w.jobQueue.Close() //nolint,gosec
		w.jobQueue = nil
	}

	if w.jobResultQueue != nil {
		w.jobResultQueue.Close() //nolint,gosec
		w.jobResultQueue = nil
	}
}
