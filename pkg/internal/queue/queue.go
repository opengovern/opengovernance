package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

const (
	prefetchCount = 10
	prefetchSize  = 0 // Disabled prefetch size
)

type Interface interface {
	Consume() (<-chan amqp.Delivery, error)
	Publish(v interface{}) error
	Close()
}

// queue is message queue based on the AMQP protocol. It uses RabbitMQ as the
// distributed system for publishing and consuming messages.
type queue struct {
	cfg  Config
	conn *amqp.Connection
	ch   *amqp.Channel
}

type Config struct {
	Server struct {
		Host     string
		Port     int
		Username string
		Password string
	}

	Queue struct {
		Name         string
		Durable      bool
		DeleteUnused bool
		Exclusive    bool
		NoWait       bool
	}

	Consumer struct {
		ID string
	}

	Producer struct {
		ID string
	}
}

func (cfg *Config) validate() error {
	switch {
	case cfg.Server.Host == "":
		return fmt.Errorf("Server.Host must be provided")
	case cfg.Server.Port == 0:
		return fmt.Errorf("Server.Port must be provided")
	case cfg.Queue.Name == "":
		return fmt.Errorf("Queue.Name must be provided")
	case cfg.Consumer.ID == "" && cfg.Producer.ID == "":
		return fmt.Errorf("cfg.Consumer.ID or cfg.Producer.ID must be provided")
	default:
		return nil
	}
}

func New(cfg Config) (Interface, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	q := &queue{cfg: cfg}

	if err := q.setup(); err != nil {
		q.Close()
		return nil, err
	}

	// Ensure Queue is declared
	_, err := q.ch.QueueDeclare(
		q.cfg.Queue.Name,         // name
		q.cfg.Queue.Durable,      // durable
		q.cfg.Queue.DeleteUnused, // delete when unused
		q.cfg.Queue.Exclusive,    // exclusive
		q.cfg.Queue.NoWait,       // no-wait
		nil,                      // arguments
	)
	if err != nil {
		q.Close()
		return nil, fmt.Errorf("creating queue: %w", err)
	}

	return q, nil
}

func (q *queue) setup() error {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		q.cfg.Server.Username,
		q.cfg.Server.Password,
		q.cfg.Server.Host,
		q.cfg.Server.Port,
	)
	conn, err := amqp.Dial(url)
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

func (q *queue) Consume() (<-chan amqp.Delivery, error) {
	return q.ch.Consume(
		q.cfg.Queue.Name,  // queue
		q.cfg.Consumer.ID, // consumer
		false,             // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)

}

func (q *queue) Publish(v interface{}) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}

	p := amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
		AppId:       q.cfg.Producer.ID,
		Timestamp:   time.Now(),
	}

	return q.ch.Publish(
		"",               // exchange
		q.cfg.Queue.Name, // routing key
		false,            // mandatory
		false,            // immediate
		p)
}

func (q *queue) Close() {
	if q.conn != nil {
		_ = q.conn.Close()
	}

	if q.ch != nil {
		_ = q.ch.Close()
	}
}
