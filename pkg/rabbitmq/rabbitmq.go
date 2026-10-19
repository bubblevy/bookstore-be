package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bubblevy/bookstore/pkg/email"
	"github.com/bubblevy/bookstore/pkg/logger"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName = "bookstore.direct"
	ExchangeType = "direct"
	EmailQueue   = "bookstore.email.queue"
	EmailRouting = "bookstore.email"
)

type Config struct {
	URL      string
	Host     string
	Port     string
	User     string
	Password string
	Enabled  bool
}

type EmailTaskType string

const (
	TaskActivationEmail    EmailTaskType = "activation_email"
	TaskResetPasswordEmail EmailTaskType = "reset_password_email"
)

type EmailPayload struct {
	TaskType      EmailTaskType `json:"task_type"`
	ToEmail       string        `json:"to_email"`
	RecipientName string        `json:"recipient_name"`
	Token         string        `json:"token"`
}

type Client interface {
	PublishEmail(ctx context.Context, payload EmailPayload) error
	StartEmailConsumer(ctx context.Context, sender email.EmailSender) error
	Close() error
	IsConnected() bool
}

type rabbitMQClient struct {
	conn   *amqp.Connection
	ch     *amqp.Channel
	cfg    Config
	mu     sync.RWMutex
	closed bool
}

func (c Config) GetURL() string {
	if c.URL != "" {
		return c.URL
	}
	host := c.Host
	if host == "" {
		host = "localhost"
	}
	port := c.Port
	if port == "" {
		port = "5672"
	}
	user := c.User
	if user == "" {
		user = "guest"
	}
	pass := c.Password
	if pass == "" {
		pass = "guest"
	}
	return fmt.Sprintf("amqp://%s:%s@%s:%s/", user, pass, host, port)
}

// NewRabbitMQClient initializes a connection to RabbitMQ, sets up the direct exchange, and asserts the durable queue.
func NewRabbitMQClient(cfg Config) (Client, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	url := cfg.GetURL()
	var conn *amqp.Connection
	var err error

	// Retry connection up to 15 times (total ~30s) to allow RabbitMQ container warm-up on cold boots
	maxRetries := 15
	for attempt := 1; attempt <= maxRetries; attempt++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		if attempt < maxRetries {
			logger.Warn("waiting for rabbitmq to become ready...", "attempt", attempt, "max", maxRetries, "error", err)
			time.Sleep(2 * time.Second)
		}
	}
	if err != nil {
		logger.Warn("failed to connect to rabbitmq after retries", "url", url, "error", err)
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open rabbitmq channel: %w", err)
	}

	// Declare direct exchange
	err = ch.ExchangeDeclare(
		ExchangeName,
		ExchangeType,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to declare rabbitmq exchange: %w", err)
	}

	// Declare durable email queue
	q, err := ch.QueueDeclare(
		EmailQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to declare rabbitmq queue: %w", err)
	}

	// Bind email queue to direct exchange
	err = ch.QueueBind(
		q.Name,
		EmailRouting,
		ExchangeName,
		false,
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to bind queue to exchange: %w", err)
	}

	logger.Info("connected to rabbitmq successfully", "queue", EmailQueue)
	return &rabbitMQClient{
		conn: conn,
		ch:   ch,
		cfg:  cfg,
	}, nil
}

func (c *rabbitMQClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil && !c.conn.IsClosed() && !c.closed
}

func (c *rabbitMQClient) PublishEmail(ctx context.Context, payload EmailPayload) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed || c.ch == nil {
		return errors.New("rabbitmq channel is closed")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	return c.ch.PublishWithContext(
		ctx,
		ExchangeName,
		EmailRouting,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // persistent message delivery
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
}

func (c *rabbitMQClient) StartEmailConsumer(ctx context.Context, sender email.EmailSender) error {
	c.mu.RLock()
	ch := c.ch
	c.mu.RUnlock()

	if ch == nil {
		return errors.New("rabbitmq channel is nil")
	}

	// Prefetch count 5 for fair queue distribution
	if err := ch.Qos(5, 0, false); err != nil {
		return fmt.Errorf("failed setting rabbitmq Qos: %w", err)
	}

	msgs, err := ch.Consume(
		EmailQueue,
		"",    // consumer tag
		false, // auto-ack: false for manual acknowledgement
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed consuming rabbitmq queue: %w", err)
	}

	go func() {
		logger.Info("rabbitmq email worker started consuming", "queue", EmailQueue)
		for {
			select {
			case <-ctx.Done():
				logger.Info("rabbitmq email worker stopped")
				return
			case d, ok := <-msgs:
				if !ok {
					logger.Warn("rabbitmq email consumer channel closed")
					return
				}

				var payload EmailPayload
				if err := json.Unmarshal(d.Body, &payload); err != nil {
					logger.Error("failed unmarshaling email payload from queue", "error", err)
					_ = d.Nack(false, false) // discard unparseable message
					continue
				}

				sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				var sendErr error
				switch payload.TaskType {
				case TaskActivationEmail:
					sendErr = sender.SendActivationEmail(sendCtx, payload.ToEmail, payload.RecipientName, payload.Token)
				case TaskResetPasswordEmail:
					sendErr = sender.SendResetPasswordEmail(sendCtx, payload.ToEmail, payload.RecipientName, payload.Token)
				default:
					logger.Warn("unrecognized email task type in queue", "type", payload.TaskType)
				}
				cancel()

				if sendErr != nil {
					logger.Error("failed executing email task from rabbitmq", "to", payload.ToEmail, "type", payload.TaskType, "error", sendErr)
					_ = d.Nack(false, true) // requeue for retry
				} else {
					logger.Info("email task processed successfully via rabbitmq", "to", payload.ToEmail, "type", payload.TaskType)
					_ = d.Ack(false)
				}
			}
		}
	}()

	return nil
}

func (c *rabbitMQClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true

	var errs []error
	if c.ch != nil {
		if err := c.ch.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing rabbitmq: %v", errs)
	}
	return nil
}
