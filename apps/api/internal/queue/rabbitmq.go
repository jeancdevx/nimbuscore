package queue

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	if err := declareTopology(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &RabbitMQ{conn: conn, channel: ch}, nil
}

func declareTopology(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		"nimbuscore.workspace",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
}

func (r *RabbitMQ) Channel() *amqp.Channel {
	return r.channel
}

func (r *RabbitMQ) NewChannel() (*amqp.Channel, error) {
	return r.conn.Channel()
}

func (r *RabbitMQ) Close() error {
	if err := r.channel.Close(); err != nil {
		return err
	}
	return r.conn.Close()
}

func (r *RabbitMQ) Ping(ctx context.Context) error {
	_, err := r.conn.Channel()
	if err != nil {
		return err
	}
	return nil
}
