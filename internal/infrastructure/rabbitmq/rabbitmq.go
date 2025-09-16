package rabbitmq

import (
	"context"
	"log"
	"core-backend/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
	Queue   amqp.Queue
}

func NewRabbitMQ(ctx context.Context) (*RabbitMQ, error) {
	cfg := config.GetAppConfig().RabbitMQ
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	queue, err := ch.QueueDeclare(
		cfg.Queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{},
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	return &RabbitMQ{Conn: conn, Channel: ch, Queue: queue}, nil
}

func (r *RabbitMQ) Publish(ctx context.Context, body []byte) error {
	return r.Channel.PublishWithContext(ctx,
		"", // exchange
		r.Queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func (r *RabbitMQ) Consume(ctx context.Context, handler func([]byte)) error {
	msgs, err := r.Channel.Consume(
		r.Queue.Name,
		"",
		true,
		false,
		false,
		false,
		amqp.Table{},
	)
	if err != nil {
		return err
	}
	go func() {
		for msg := range msgs {
			handler(msg.Body)
		}
	}()
	return nil
}

func (r *RabbitMQ) Close() {
	if r.Channel != nil {
		r.Channel.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}
