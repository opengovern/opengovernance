package describe

import (
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"time"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"go.uber.org/zap"
)

type esResource struct {
	index string
	id    string
	body  []byte
}

type KafkaEsSink struct {
	logger        *zap.Logger
	kafkaConsumer *confluent_kafka.Consumer
	esClient      keibi.Client
	commitChan    chan *confluent_kafka.Message
	esSinkChan    chan *confluent_kafka.Message
	esSinkBuffer  []*confluent_kafka.Message
}

func NewKafkaEsSink(logger *zap.Logger, kafkaConsumer *confluent_kafka.Consumer, esClient keibi.Client) *KafkaEsSink {
	return &KafkaEsSink{
		logger:        logger,
		kafkaConsumer: kafkaConsumer,
		esClient:      esClient,
		commitChan:    make(chan *confluent_kafka.Message, 5000),
		esSinkChan:    make(chan *confluent_kafka.Message, 5000),
		esSinkBuffer:  nil,
	}
}

func (s *KafkaEsSink) Run() {
	EnsureRunGoroutin(func() {
		s.runKafkaRead()
	})
	EnsureRunGoroutin(func() {
		s.runKafkaCommit()
	})
	EnsureRunGoroutin(func() {
		s.runElasticSearchSink()
	})
}

func (s *KafkaEsSink) runElasticSearchSink() {
	for {
		select {
		case resource := <-s.esSinkChan:
			s.esSinkBuffer = append(s.esSinkBuffer, resource)
			if len(s.esSinkBuffer) > 5000 {
				s.flushESSinkBuffer()
			}
		case <-time.After(30 * time.Second):
			s.flushESSinkBuffer()
		}
	}
}

func (s *KafkaEsSink) runKafkaRead() {
	for {
		ev, err := s.kafkaConsumer.ReadMessage(time.Millisecond * 100)
		if err != nil {
			if err.Error() == confluent_kafka.ErrTimedOut.String() {
				continue
			}
			s.logger.Error("Failed to read kafka message", zap.Error(err))
			continue
		}
		if ev == nil {
			continue
		}
		s.esSinkChan <- ev
	}
}

func (s *KafkaEsSink) runKafkaCommit() {
	for msg := range s.commitChan {
		_, err := s.kafkaConsumer.CommitMessage(msg)
		if err != nil {
			s.logger.Error("Failed to commit kafka message", zap.Error(err))
		}
	}
}

func (s *KafkaEsSink) flushESSinkBuffer() {
	if len(s.esSinkBuffer) == 0 {
		return
	}

	//esMsgs := make([]*esResource, 0, len(s.esSinkBuffer))
	//for _, msg := range s.esSinkBuffer {
	//	resource, err := newEsResourceFromKafkaMessage(msg)
	//	if err != nil || resourceEvent == nil {
	//		s.logger.Error("Failed to parse kafka message", zap.Error(err))
	//		continue
	//	}
	//	esMsgs = append(esMsgs, resource)
	//}
	//TODO Send to ES

	for _, resourceEvent := range s.esSinkBuffer {
		s.commitChan <- resourceEvent
	}

	s.esSinkBuffer = nil
}

func newEsResourceFromKafkaMessage(msg *confluent_kafka.Message) (*esResource, error) {
	var resource esResource
	index := ""
	for _, h := range msg.Headers {
		if h.Key == kafka.EsIndexHeader {
			index = string(h.Value)
		}
	}
	if index == "" {
		return nil, fmt.Errorf("missing index header")
	}
	resource.index = index
	resource.id = string(msg.Key)
	resource.body = msg.Value

	return &resource, nil
}
