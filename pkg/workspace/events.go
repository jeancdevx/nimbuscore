package workspace

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type EventType string

const (
	EventWorkspaceCreate   EventType = "workspace.create"
	EventWorkspaceStart    EventType = "workspace.start"
	EventWorkspaceStop     EventType = "workspace.stop"
	EventWorkspaceDelete   EventType = "workspace.delete"
	EventWorkspaceBuild    EventType = "workspace.build"
	EventWorkspaceSnapshot EventType = "workspace.snapshot"
	EventWorkspaceTimeout  EventType = "workspace.timeout"
)

type Event struct {
	ID          uuid.UUID       `json:"id"`
	Type        EventType       `json:"type"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	UserID      uuid.UUID       `json:"user_id"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
}

type Publisher struct {
	channel  *amqp.Channel
	exchange string
}

func NewPublisher(ch *amqp.Channel, exchange string) *Publisher {
	return &Publisher{channel: ch, exchange: exchange}
}

func (p *Publisher) Publish(ctx context.Context, event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(ctx, p.exchange, string(event.Type), false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		Timestamp:    event.Timestamp,
		DeliveryMode: amqp.Persistent,
	})
}

type HandlerFunc func(ctx context.Context, event Event) error

type Consumer struct {
	channel    *amqp.Channel
	exchange   string
	queueName  string
	handlers   map[EventType]HandlerFunc
}

func NewConsumer(ch *amqp.Channel, exchange, queueName string) *Consumer {
	return &Consumer{
		channel:   ch,
		exchange:  exchange,
		queueName: queueName,
		handlers:  make(map[EventType]HandlerFunc),
	}
}

func (c *Consumer) Handle(eventType EventType, handler HandlerFunc) {
	c.handlers[eventType] = handler
}

func (c *Consumer) Start(ctx context.Context) error {
	_, err := c.channel.QueueDeclare(c.queueName, true, false, false, false, nil)
	if err != nil {
		return err
	}

	for eventType := range c.handlers {
		if err := c.channel.QueueBind(c.queueName, string(eventType), c.exchange, false, nil); err != nil {
			return err
		}
	}

	msgs, err := c.channel.ConsumeWithContext(ctx, c.queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	for msg := range msgs {
		var event Event
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			msg.Nack(false, false)
			continue
		}

		handler, ok := c.handlers[event.Type]
		if !ok {
			msg.Nack(false, false)
			continue
		}

		if err := handler(ctx, event); err != nil {
			msg.Nack(false, true)
			continue
		}

		msg.Ack(false)
	}

	return nil
}
