package producer

import "gopkg.in/Shopify/sarama.v1"

type InMemorySaramaProducer struct {
	Messages []*sarama.ProducerMessage
}

func NewInMemorySaramaProducer() *InMemorySaramaProducer {
	return &InMemorySaramaProducer{
		Messages: make([]*sarama.ProducerMessage, 0),
	}
}

func (k *InMemorySaramaProducer) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	if msg == nil {
		return 0, 0, nil
	}
	k.Messages = append(k.Messages, msg)
	return 0, 0, nil
}

func (k *InMemorySaramaProducer) SendMessages(msgs []*sarama.ProducerMessage) error {
	for _, msg := range msgs {
		k.SendMessage(msg)
	}
	return nil
}

func (k *InMemorySaramaProducer) Close() error {
	return nil
}

func (k *InMemorySaramaProducer) GetMessages() []*sarama.ProducerMessage {
	return k.Messages
}
