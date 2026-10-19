package test

import (
	"context"
	"testing"

	"github.com/bubblevy/bookstore/pkg/email"
	"github.com/bubblevy/bookstore/pkg/rabbitmq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockEmailSender struct {
	mock.Mock
}

func (m *MockEmailSender) SendActivationEmail(ctx context.Context, toEmail, recipientName, token string) error {
	args := m.Called(ctx, toEmail, recipientName, token)
	return args.Error(0)
}

func (m *MockEmailSender) SendResetPasswordEmail(ctx context.Context, toEmail, recipientName, token string) error {
	args := m.Called(ctx, toEmail, recipientName, token)
	return args.Error(0)
}

type MockRabbitMQClient struct {
	mock.Mock
}

func (m *MockRabbitMQClient) PublishEmail(ctx context.Context, payload rabbitmq.EmailPayload) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

func (m *MockRabbitMQClient) StartEmailConsumer(ctx context.Context, sender email.EmailSender) error {
	args := m.Called(ctx, sender)
	return args.Error(0)
}

func (m *MockRabbitMQClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRabbitMQClient) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestRabbitMQ_Disabled(t *testing.T) {
	cfg := rabbitmq.Config{
		Enabled: false,
	}
	client, err := rabbitmq.NewRabbitMQClient(cfg)
	assert.NoError(t, err)
	assert.Nil(t, client)
}

func TestRabbitMQEmailSender(t *testing.T) {
	ctx := context.Background()

	t.Run("publishes to rabbitmq when connected", func(t *testing.T) {
		mockClient := new(MockRabbitMQClient)
		mockFallback := new(MockEmailSender)

		sender := rabbitmq.NewRabbitMQEmailSender(mockClient, mockFallback)

		mockClient.On("IsConnected").Return(true)
		mockClient.On("PublishEmail", ctx, rabbitmq.EmailPayload{
			TaskType:      rabbitmq.TaskActivationEmail,
			ToEmail:       "user@example.com",
			RecipientName: "John Doe",
			Token:         "token-123",
		}).Return(nil).Once()

		err := sender.SendActivationEmail(ctx, "user@example.com", "John Doe", "token-123")
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
		mockFallback.AssertNotCalled(t, "SendActivationEmail")
	})

	t.Run("falls back to direct SMTP when rabbitmq client is disconnected", func(t *testing.T) {
		mockClient := new(MockRabbitMQClient)
		mockFallback := new(MockEmailSender)

		sender := rabbitmq.NewRabbitMQEmailSender(mockClient, mockFallback)

		mockClient.On("IsConnected").Return(false)
		mockFallback.On("SendActivationEmail", ctx, "user@example.com", "John Doe", "token-123").Return(nil).Once()

		err := sender.SendActivationEmail(ctx, "user@example.com", "John Doe", "token-123")
		assert.NoError(t, err)
		mockFallback.AssertExpectations(t)
		mockClient.AssertNotCalled(t, "PublishEmail")
	})

	t.Run("publishes reset password email when connected", func(t *testing.T) {
		mockClient := new(MockRabbitMQClient)
		mockFallback := new(MockEmailSender)

		sender := rabbitmq.NewRabbitMQEmailSender(mockClient, mockFallback)

		mockClient.On("IsConnected").Return(true)
		mockClient.On("PublishEmail", ctx, rabbitmq.EmailPayload{
			TaskType:      rabbitmq.TaskResetPasswordEmail,
			ToEmail:       "user@example.com",
			RecipientName: "John Doe",
			Token:         "reset-456",
		}).Return(nil).Once()

		err := sender.SendResetPasswordEmail(ctx, "user@example.com", "John Doe", "reset-456")
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
		mockFallback.AssertNotCalled(t, "SendResetPasswordEmail")
	})
}
