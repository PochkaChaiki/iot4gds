package queue

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func NewRabbitConnection(uri string) (*amqp.Connection, error) {
	return amqp.Dial(uri)
}

func DeclareQueue(ch *amqp.Channel, name string) error {
	_, err := ch.QueueDeclare(
		name,
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}
