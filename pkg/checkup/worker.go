package checkup

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kaytu-io/kaytu-engine/pkg/jq"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/zap"
)

type Worker struct {
	id            string
	jq            *jq.JobQueue
	logger        *zap.Logger
	pusher        *push.Pusher
	onboardClient client.OnboardServiceClient
}

func NewWorker(
	id string,
	natsURL string,
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

	jq, err := jq.New(natsURL, logger)
	if err != nil {
		return nil, err
	}
	w.jq = jq

	w.logger = logger

	w.pusher = push.New(prometheusPushAddress, "checkup-worker")
	w.pusher.Collector(DoCheckupJobsCount).
		Collector(DoCheckupJobsDuration)

	w.onboardClient = client.NewOnboardServiceClient(onboardBaseURL)
	return w, nil
}

func (w *Worker) Run() error {
	ctx := context.Background()

	if _, err := w.jq.Consume(
		ctx,
		"checkup-service",
		StreamName,
		[]string{JobsQueueName},
		"checkup-service",
		func(msg jetstream.Msg) {
			var job Job
			if err := json.Unmarshal(msg.Data(), &job); err != nil {
				w.logger.Error("Failed to unmarshal task", zap.Error(err))

				// sending ack for message because we cannot do anything
				// more by repeating the process.
				if err = msg.Ack(); err != nil {
					w.logger.Error("Failed to ack the message", zap.Error(err))
				}

				return
			}

			w.logger.Info("Processing job", zap.Int("jobID", int(job.JobID)))

			result := job.Do(w.onboardClient, w.logger)

			bytes, err := json.Marshal(result)
			if err != nil {
				return
			}

			w.logger.Info("Publishing job result", zap.Int("jobID", int(job.JobID)))

			if err := w.jq.Produce(context.Background(), ResultsQueueName, bytes, fmt.Sprintf("job-%d", result.JobID)); err != nil {
				w.logger.Error("Failed to send results to queue: %s", zap.Error(err))
			}

			if err := msg.Ack(); err != nil {
				w.logger.Error("Failed to ack the message", zap.Error(err))
			}

			err = w.pusher.Push()
			if err != nil {
				w.logger.Error("Failed to push metrics", zap.Error(err))
			}
		},
	); err != nil {
		return err
	}

	w.logger.Error("Waiting indefinitely for messages. To exit press CTRL+C")
	<-ctx.Done()
	return nil
}

func (w *Worker) Stop() {
	w.pusher.Push()
}
