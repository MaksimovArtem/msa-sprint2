package kafka

import (
	"encoding/json"

	"github.com/IBM/sarama"
)

type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewProducer(brokers []string) (*Producer, error) {
	return NewProducerWithTopic(brokers, "booking-events")
}

func NewProducerWithTopic(brokers []string, topic string) (*Producer, error) {

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	// Wait for all in-sync replicas to ack the message.
	// This is the strongest delivery confirmation you can get from Kafka on produce.
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3

	p, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return &Producer{producer: p, topic: topic}, nil
}

func (p *Producer) SendBookingCreated(event interface{}) (partition int32, offset int64, err error) {

	data, err := json.Marshal(event)
	if err != nil {
		return 0, 0, err
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Value: sarama.ByteEncoder(data),
	}

	partition, offset, err = p.producer.SendMessage(msg)
	return partition, offset, err
}

func (p *Producer) Close() error {
	return p.producer.Close()
}

func (p *Producer) Topic() string {
	return p.topic
}
