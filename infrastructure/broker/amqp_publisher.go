package broker

import (
	"context"
	"errors"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AMQPPublisher struct {
	amqpURL  string
	exchange string

	mu   sync.Mutex
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewAMQPPublisher(amqpURL, exchange string) *AMQPPublisher {
	return &AMQPPublisher{amqpURL: amqpURL, exchange: exchange}
}

func (p *AMQPPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error
	if p.ch != nil {
		err = p.ch.Close()
		p.ch = nil
	}
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}
	return err
}

func (p *AMQPPublisher) ensureConnected() error {
	if p.ch != nil && p.conn != nil && !p.conn.IsClosed() {
		return nil
	}
	if p.amqpURL == "" {
		return errors.New("missing amqp url")
	}

	conn, err := amqp.Dial(p.amqpURL)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}

	if err := ch.ExchangeDeclare(
		p.exchange,
		"topic",
		true,  // durable
		false, // autoDelete
		false, // internal
		false, // noWait
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}

	p.conn = conn
	p.ch = ch
	return nil
}

func (p *AMQPPublisher) Publish(ctx context.Context, routingKey string, body []byte, headers amqp.Table, timeout time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.ensureConnected(); err != nil {
		return err
	}

	pubCtx := ctx
	var cancel func()
	if timeout > 0 {
		pubCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	return p.ch.PublishWithContext(
		pubCtx,
		p.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now().UTC(),
			Headers:      headers,
			Body:         body,
		},
	)
}
