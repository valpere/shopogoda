//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valpere/shopogoda/internal/bot"
	"github.com/valpere/shopogoda/internal/config"
)

func TestBotEndToEnd(t *testing.T) {
	// Skip if no bot token provided
	botToken := os.Getenv("TEST_TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		t.Skip("No TEST_TELEGRAM_BOT_TOKEN provided, skipping E2E tests")
	}

	// Load test configuration
	cfg := &config.Config{
		Bot: config.BotConfig{
			Token:       botToken,
			Debug:       true,
			WebhookPort: 8081,
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "weather_user",
			Password: "weather_pass",
			Name:     "weather_bot_test",
			SSLMode:  "disable",
		},
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
			DB:   1, // Use different DB for tests
		},
		Weather: config.WeatherConfig{
			OpenWeatherAPIKey: os.Getenv("TEST_OPENWEATHER_API_KEY"),
		},
	}

	// Create bot instance
	weatherBot, err := bot.New(cfg)
	require.NoError(t, err)

	// Start bot
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := weatherBot.Start(ctx)
		if err != nil {
			t.Errorf("Bot failed to start: %v", err)
		}
	}()

	// Wait for bot to start
	time.Sleep(2 * time.Second)

	// Test bot info
	botAPI, err := gotgbot.NewBot(botToken, nil)
	require.NoError(t, err)

	me, err := botAPI.GetMe()
	require.NoError(t, err)
	assert.NotEmpty(t, me.Username)
	assert.True(t, me.IsBot)

	// Stop bot
	err = weatherBot.Stop()
	assert.NoError(t, err)
}
