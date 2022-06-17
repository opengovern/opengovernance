package queue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
)

type QueueTestSuite struct {
	suite.Suite

	rabbit dockertest.RabbitMQServer
}

func TestQueue(t *testing.T) {
	suite.Run(t, &QueueTestSuite{})
}

func (ts *QueueTestSuite) SetupSuite() {
	ts.rabbit = dockertest.StartupRabbitMQ(ts.T())
}

func (ts *QueueTestSuite) TestRabbitMQ() {
	cfg := Config{
		Server: ts.rabbit,
	}

	cfg.Queue.Name = "test-queue"
	cfg.Consumer.ID = "test-consumer"
	cfg.Producer.ID = "test-producer"

	qu, err := New(cfg)
	ts.NoError(err, "create queue")

	type Message struct {
		ID uint
	}

	msgs, err := qu.Consume()
	ts.NoError(err, "consume")

	// try to retrieve one without any message
	select {
	case msg := <-msgs:
		ts.FailNow("unexpected message: %#v", msg)
	default:
	}

	err = qu.Publish(Message{
		ID: 1,
	})
	ts.NoError(err, "publish")

	select {
	case msg := <-msgs:
		ts.Equal("application/json", msg.ContentType)
		ts.Equal("test-queue", msg.RoutingKey)
		ts.Equal("test-producer", msg.AppId)
		ts.Equal("test-consumer", msg.ConsumerTag)
		ts.Equal("", msg.Expiration)
		ts.Equal(`{"ID":1}`, string(msg.Body))
		err := msg.Ack(false)
		ts.NoError(err, "ack")
	case <-time.After(10 * time.Second):
		ts.FailNow("timed out: exptected message with id 1")
	}

	// try to retrieve another one
	select {
	case msg := <-msgs:
		ts.FailNow("unexpected message: %#v", msg)
	default:
	}
}

func (ts *QueueTestSuite) TestRabbitMQGetQueueInfo() {
	cfg := Config{
		Server: ts.rabbit,
	}

	cfg.Queue.Name = "test-queue-1"
	cfg.Consumer.ID = "test-consumer"
	cfg.Producer.ID = "test-producer"

	queue, err := New(cfg)
	ts.NoError(err, "create queue")
	// new queue with no data
	count, err := queue.Len()
	ts.NoError(err, "get queue length")
	ts.Equal(0, count)
	ts.Equal(cfg.Queue.Name, queue.Name())

	type Message struct {
		ID uint
	}

	msgs, err := queue.Consume()
	ts.NoError(err, "consume")

	// try to retrieve one without any message
	select {
	case msg := <-msgs:
		ts.FailNow("unexpected message: %#v", msg)
	default:
	}

	err = queue.Publish(Message{
		ID: 1,
	})
	ts.NoError(err, "publish")

	select {
	case msg := <-msgs:
		ts.Equal("application/json", msg.ContentType)
		ts.Equal(queue.Name(), msg.RoutingKey)
		ts.Equal("test-producer", msg.AppId)
		ts.Equal("test-consumer", msg.ConsumerTag)
		ts.Equal("", msg.Expiration)
		ts.Equal(`{"ID":1}`, string(msg.Body))
		err := msg.Ack(false)
		ts.NoError(err, "ack")
	case <-time.After(10 * time.Second):
		ts.FailNow("timed out: exptected message with id 1")
	}

	// try to retrieve another one
	select {
	case msg := <-msgs:
		ts.FailNow("unexpected message: %#v", msg)
	default:
	}
}
