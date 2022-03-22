package queue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
)

func TestRabbitMQ(t *testing.T) {
	require := require.New(t)
	server := dockertest.StartupRabbitMQ(t)

	cfg := Config{
		Server: server,
	}

	cfg.Queue.Name = "test-queue"
	cfg.Consumer.ID = "test-consumer"
	cfg.Producer.ID = "test-producer"

	qu, err := New(cfg)
	require.NoError(err, "create queue")

	type Message struct {
		ID uint
	}

	msgs, err := qu.Consume()
	require.NoError(err, "consume")

	// Try to retrieve one without any message
	select {
	case msg := <-msgs:
		require.FailNow("unexpected message: %#v", msg)
	default:
		// pass
	}

	err = qu.Publish(Message{
		ID: 1,
	})
	require.NoError(err, "publish")

	select {
	case msg := <-msgs:
		require.Equal("application/json", msg.ContentType)
		require.Equal("test-queue", msg.RoutingKey)
		require.Equal("test-producer", msg.AppId)
		require.Equal("test-consumer", msg.ConsumerTag)
		require.Equal("", msg.Expiration)
		require.Equal(`{"ID":1}`, string(msg.Body))
		err := msg.Ack(false)
		require.NoError(err, "ack")
	case <-time.After(10 * time.Second):
		require.FailNow("timed out: exptected message with id 1")
	}

	// Try to retrieve another one
	select {
	case msg := <-msgs:
		require.FailNow("unexpected message: %#v", msg)
	default:
		// pass
	}
}
