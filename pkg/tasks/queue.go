package tasks

import (
	"fmt"

	"github.com/streadway/amqp"
)

const (
	prefetchCount     = 10
	prefetchSize      = 0 // Disabled prefetch size
	describeQueueName = "describe-queue"
)

// Queue is message queue based on the AMQP protocol. It uses RabbitMQ as the
// distributed system for publishing and consuming messages.
type Queue struct {
	url  string
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewDescribeQueue(url string) (q *Queue, err error) {
	// Close underlying connections if anything fails along the way
	defer func() {
		if err != nil && q != nil {
			q.Close()
		}
	}()

	q = &Queue{
		url: url,
	}

	if err := q.setup(); err != nil {
		return nil, err
	}

	// Ensure Queue is declared
	_, err = q.ch.QueueDeclare(
		describeQueueName, // name
		false,             // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("creating queue: %w", err)
	}

	return q, nil
}

func (q *Queue) setup() error {
	conn, err := amqp.Dial(q.url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("creating channel: %w", err)
	}

	err = ch.Qos(prefetchCount, prefetchSize, false)
	if err != nil {
		return fmt.Errorf("setting prefetch attributes: %w", err)
	}

	q.conn = conn
	q.ch = ch
	return nil
}

func (q *Queue) Consume(consumer string) (<-chan amqp.Delivery, error) {
	return q.ch.Consume(
		describeQueueName, // queue
		consumer,          // consumer
		false,             // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
}

func (q *Queue) Publish(p amqp.Publishing) error {
	return q.ch.Publish(
		"",                // exchange
		describeQueueName, // routing key
		false,             // mandatory
		false,             // immediate
		p)
}

func (q *Queue) Close() {
	if q.conn != nil {
		_ = q.conn.Close()
	}

	if q.ch != nil {
		_ = q.ch.Close()
	}
}
