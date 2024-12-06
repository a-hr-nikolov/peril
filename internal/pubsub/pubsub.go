package pubsub

import (
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	jsonData, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("could not marshal value %v:\n%w", val, err)
	}

	err = ch.Publish(
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonData,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"could not publish to exchange %v with key %v:\n%w",
			exchange, key, err,
		)
	}

	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	durable bool,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not create channel: %w", err)
	}

	queue, err := ch.QueueDeclare(queueName, durable, !durable, !durable, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not create queue: %w", err)
	}
	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not bind queue: %w", err)
	}

	return ch, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	durable bool,
	handler func(T) AckType,
) error {
	ch, _, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		durable,
	)
	if err != nil {
		return fmt.Errorf("could not declare and bind queue: %w", err)
	}

	deliveryCh, err := ch.Consume(
		queueName, "", false, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("could not consume queue: %w", err)
	}

	go func() {
		for delivery := range deliveryCh {
			var data T
			err := json.Unmarshal(delivery.Body, &data)
			if err != nil {
				log.Printf("could not unmarshal delivery: %v", err)
			}
			ackType := handler(data)
			switch ackType {
			case Ack:
				delivery.Ack(false)
			case NackRequeue:
				delivery.Nack(false, true)
			case NackDiscard:
				delivery.Nack(false, false)
			}
		}
	}()

	return nil
}
