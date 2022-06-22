package kafka

import (
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
)

func DoSendToKafka(producer sarama.SyncProducer, topic string, kafkaMsgs []InsightResource, logger *zap.Logger) error {
	var msgs []*sarama.ProducerMessage
	for _, v := range kafkaMsgs {
		msg, err := v.AsProducerMessage()
		if err != nil {
			logger.Error("Failed calling AsProducerMessage", zap.Error(fmt.Errorf("Failed to convert msg[%s] to Kafka ProducerMessage, ignoring...", v.MessageID())))
			continue
		}

		// Override the topic
		msg.Topic = topic

		msgs = append(msgs, msg)
	}

	if len(msgs) == 0 {
		return nil
	}

	if err := producer.SendMessages(msgs); err != nil {
		if errs, ok := err.(sarama.ProducerErrors); ok {
			for _, e := range errs {
				logger.Error("Falied calling SendMessages", zap.Error(fmt.Errorf("Failed to persist resource[%s] in kafka topic[%s]: %s\nMessage: %v\n", e.Msg.Key, e.Msg.Topic, e.Error(), e.Msg)))
			}
		}

		return err
	}

	return nil
}
