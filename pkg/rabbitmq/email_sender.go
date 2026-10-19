package rabbitmq

import (
	"context"

	"github.com/bubblevy/bookstore/pkg/email"
	"github.com/bubblevy/bookstore/pkg/logger"
)

type rabbitMQEmailSender struct {
	client   Client
	fallback email.EmailSender
}

// NewRabbitMQEmailSender returns an email.EmailSender implementation that routes emails through RabbitMQ.
// If RabbitMQ is unavailable or fails, it gracefully falls back to direct SMTP.
func NewRabbitMQEmailSender(client Client, fallback email.EmailSender) email.EmailSender {
	return &rabbitMQEmailSender{
		client:   client,
		fallback: fallback,
	}
}

func (s *rabbitMQEmailSender) SendActivationEmail(ctx context.Context, toEmail, recipientName, activationToken string) error {
	if s.client != nil && s.client.IsConnected() {
		err := s.client.PublishEmail(ctx, EmailPayload{
			TaskType:      TaskActivationEmail,
			ToEmail:       toEmail,
			RecipientName: recipientName,
			Token:         activationToken,
		})
		if err == nil {
			logger.Info("enqueued activation email to rabbitmq queue", "to", toEmail)
			return nil
		}
		logger.Warn("failed enqueuing activation email to rabbitmq, falling back to direct smtp", "error", err)
	}

	if s.fallback != nil {
		return s.fallback.SendActivationEmail(ctx, toEmail, recipientName, activationToken)
	}
	return nil
}

func (s *rabbitMQEmailSender) SendResetPasswordEmail(ctx context.Context, toEmail, recipientName, resetToken string) error {
	if s.client != nil && s.client.IsConnected() {
		err := s.client.PublishEmail(ctx, EmailPayload{
			TaskType:      TaskResetPasswordEmail,
			ToEmail:       toEmail,
			RecipientName: recipientName,
			Token:         resetToken,
		})
		if err == nil {
			logger.Info("enqueued reset password email to rabbitmq queue", "to", toEmail)
			return nil
		}
		logger.Warn("failed enqueuing reset password email to rabbitmq, falling back to direct smtp", "error", err)
	}

	if s.fallback != nil {
		return s.fallback.SendResetPasswordEmail(ctx, toEmail, recipientName, resetToken)
	}
	return nil
}
