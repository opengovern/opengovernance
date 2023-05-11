package kafka

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
)

const (
	EsIndexHeader = "elasticsearch_index"
)

type Doc interface {
	KeysAndIndex() ([]string, string)
}

func trimEmptyMaps(input map[string]interface{}) {
	for key, value := range input {
		switch value.(type) {
		case map[string]interface{}:
			if len(value.(map[string]interface{})) != 0 {
				trimEmptyMaps(value.(map[string]interface{}))
			}
			if len(value.(map[string]interface{})) == 0 {
				delete(input, key)
			}
		}
	}
}

func trimJsonFromEmptyObjects(input []byte) ([]byte, error) {
	unknownData := map[string]interface{}{}
	err := json.Unmarshal(input, &unknownData)
	if err != nil {
		return nil, err
	}
	trimEmptyMaps(unknownData)
	return json.Marshal(unknownData)
}

func asProducerMessage(r Doc) (*sarama.ProducerMessage, error) {
	keys, index := r.KeysAndIndex()
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	value, err = trimJsonFromEmptyObjects(value)
	if err != nil {
		return nil, err
	}

	return Msg(HashOf(keys...), value, index), nil
}

func messageID(r Doc) string {
	k, _ := r.KeysAndIndex()
	return fmt.Sprintf("%v", k)
}

func HashOf(strings ...string) string {
	h := sha256.New()
	for _, s := range strings {
		h.Write([]byte(s))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func Msg(key string, value []byte, index string) *sarama.ProducerMessage {
	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(key),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(EsIndexHeader),
				Value: []byte(index),
			},
		},
		Value: sarama.ByteEncoder(value),
	}
}

func DoSend(producer sarama.SyncProducer, topic string, partition int32, docs []Doc, logger *zap.Logger) error {
	var msgs []*sarama.ProducerMessage
	for _, v := range docs {
		msg, err := asProducerMessage(v)
		if err != nil {
			logger.Error("Failed calling AsProducerMessage", zap.Error(fmt.Errorf("Failed to convert msg[%s] to Kafka ProducerMessage, ignoring...", messageID(v))))
			continue
		}

		// Override the topic
		msg.Topic = topic
		msg.Partition = partition

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
