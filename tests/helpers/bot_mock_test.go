package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMockBot(t *testing.T) {
	mockBot := NewMockBot()

	assert.NotNil(t, mockBot)
	assert.NotNil(t, mockBot.Bot)
	assert.Equal(t, int64(12345), mockBot.Bot.Id)
	assert.True(t, mockBot.Bot.IsBot)
	assert.Equal(t, "TestBot", mockBot.Bot.FirstName)
	assert.Equal(t, "test_bot", mockBot.Bot.Username)
	assert.Equal(t, "test_token", mockBot.Bot.Token)
}

func TestNewMockContext(t *testing.T) {
	t.Run("minimal context", func(t *testing.T) {
		mockCtx := NewMockContext(MockContextOptions{})

		assert.NotNil(t, mockCtx)
		assert.NotNil(t, mockCtx.Context)
		assert.Equal(t, int64(12345), mockCtx.Context.EffectiveUser.Id)
		assert.Equal(t, "testuser", mockCtx.Context.EffectiveUser.Username)
		assert.Equal(t, "Test", mockCtx.Context.EffectiveUser.FirstName)
		assert.NotNil(t, mockCtx.Context.Data)
	})

	t.Run("with custom user ID", func(t *testing.T) {
		mockCtx := NewMockContext(MockContextOptions{
			UserID: 99999,
		})

		assert.Equal(t, int64(99999), mockCtx.Context.EffectiveUser.Id)
	})

	t.Run("with args", func(t *testing.T) {
		args := []string{"/weather", "New", "York"}
		mockCtx := NewMockContext(MockContextOptions{
			Args: args,
		})

		assert.Equal(t, "/weather New York", mockCtx.Context.EffectiveMessage.Text)
		assert.Equal(t, "/weather New York", mockCtx.Context.Update.Message.Text)
	})

	t.Run("with location", func(t *testing.T) {
		mockCtx := NewMockContext(MockContextOptions{
			Latitude:  40.7128,
			Longitude: -74.0060,
		})

		assert.NotNil(t, mockCtx.Context.EffectiveMessage.Location)
		assert.Equal(t, 40.7128, mockCtx.Context.EffectiveMessage.Location.Latitude)
		assert.Equal(t, -74.0060, mockCtx.Context.EffectiveMessage.Location.Longitude)
	})

	t.Run("with callback query", func(t *testing.T) {
		mockCtx := NewMockContext(MockContextOptions{
			CallbackID: "callback_123",
			Data:       "action:confirm",
		})

		assert.NotNil(t, mockCtx.Context.CallbackQuery)
		assert.Equal(t, "callback_123", mockCtx.Context.CallbackQuery.Id)
		assert.Equal(t, "action:confirm", mockCtx.Context.CallbackQuery.Data)
	})
}

func TestNewSimpleMockContext(t *testing.T) {
	mockCtx := NewSimpleMockContext(12345, "Hello, world!")

	assert.NotNil(t, mockCtx)
	assert.Equal(t, int64(12345), mockCtx.Context.EffectiveUser.Id)
	assert.Equal(t, "Hello, world!", mockCtx.Context.EffectiveMessage.Text)
}

func TestNewMockContextWithLocation(t *testing.T) {
	mockCtx := NewMockContextWithLocation(12345, 51.5074, -0.1278)

	assert.NotNil(t, mockCtx.Context.EffectiveMessage.Location)
	assert.Equal(t, 51.5074, mockCtx.Context.EffectiveMessage.Location.Latitude)
	assert.Equal(t, -0.1278, mockCtx.Context.EffectiveMessage.Location.Longitude)
}

func TestNewMockContextWithCallback(t *testing.T) {
	mockCtx := NewMockContextWithCallback(12345, "cb_456", "button:clicked")

	assert.NotNil(t, mockCtx.Context.CallbackQuery)
	assert.Equal(t, "cb_456", mockCtx.Context.CallbackQuery.Id)
	assert.Equal(t, "button:clicked", mockCtx.Context.CallbackQuery.Data)
}

func TestMockContextDefaults(t *testing.T) {
	t.Run("default values are set", func(t *testing.T) {
		mockCtx := NewMockContext(MockContextOptions{})

		// Check default user ID
		assert.Equal(t, int64(12345), mockCtx.Context.EffectiveUser.Id)

		// Check default username
		assert.Equal(t, "testuser", mockCtx.Context.EffectiveUser.Username)

		// Check default first name
		assert.Equal(t, "Test", mockCtx.Context.EffectiveUser.FirstName)

		// Check default chat ID
		assert.Equal(t, int64(12345), mockCtx.Context.EffectiveChat.Id)

		// Check default message ID
		assert.Equal(t, int64(1), mockCtx.Context.EffectiveMessage.MessageId)

		// Check chat type
		assert.Equal(t, "private", mockCtx.Context.EffectiveChat.Type)

		// Check user is not a bot
		assert.False(t, mockCtx.Context.EffectiveUser.IsBot)
	})

	t.Run("custom values override defaults", func(t *testing.T) {
		mockCtx := NewMockContext(MockContextOptions{
			UserID:      99999,
			Username:    "customuser",
			FirstName:   "Custom",
			LastName:    "User",
			ChatID:      88888,
			MessageID:   777,
			MessageText: "Custom message",
		})

		assert.Equal(t, int64(99999), mockCtx.Context.EffectiveUser.Id)
		assert.Equal(t, "customuser", mockCtx.Context.EffectiveUser.Username)
		assert.Equal(t, "Custom", mockCtx.Context.EffectiveUser.FirstName)
		assert.Equal(t, "User", mockCtx.Context.EffectiveUser.LastName)
		assert.Equal(t, int64(88888), mockCtx.Context.EffectiveChat.Id)
		assert.Equal(t, int64(777), mockCtx.Context.EffectiveMessage.MessageId)
		assert.Equal(t, "Custom message", mockCtx.Context.EffectiveMessage.Text)
	})
}
