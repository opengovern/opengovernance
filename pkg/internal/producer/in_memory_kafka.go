package producer

import "gopkg.in/Shopify/sarama.v1"

type InMemorySaramaProducer struct {
	Messages map[string][]*sarama.ProducerMessage
}

func NewInMemorySaramaProducer() *InMemorySaramaProducer {
	return &InMemorySaramaProducer{
		Messages: make(map[string][]*sarama.ProducerMessage),
	}
}

func (k *InMemorySaramaProducer) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	if msg == nil {
		return 0, 0, nil
	}
	if k.Messages[msg.Topic] == nil {
		k.Messages[msg.Topic] = make([]*sarama.ProducerMessage, 0)
	}
	k.Messages[msg.Topic] = append(k.Messages[msg.Topic], msg)
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

func (k *InMemorySaramaProducer) GetMessages(topic string) []*sarama.ProducerMessage {
	if k.Messages[topic] == nil {
		k.Messages[topic] = make([]*sarama.ProducerMessage, 0)
	}
	return k.Messages[topic]
}
