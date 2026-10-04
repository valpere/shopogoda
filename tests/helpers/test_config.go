package helpers

import (
	"os"

	"github.com/valpere/shopogoda/internal/config"
)

// GetTestConfig returns a configuration suitable for testing
func GetTestConfig() *config.Config {
	return &config.Config{
		Bot: config.BotConfig{
			Token: "test_bot_token", // #nosec G101
			Debug: true,
		},
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		Weather: config.WeatherConfig{
			OpenWeatherAPIKey: "test_weather_api_key",
			UserAgent:         "ShoPogoda-Weather-Bot/1.0 (test@shopogoda.bot)",
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "console",
		},
		Metrics: config.MetricsConfig{
			Port: 2113, // Different port for tests
		},
		Integrations: config.IntegrationsConfig{
			SlackWebhookURL: "https://hooks.slack.com/test",
			GrafanaURL:      "http://localhost:3001",
		},
	}
}

// GetTestConfigFromEnv returns test config with environment overrides
func GetTestConfigFromEnv() *config.Config {
	cfg := GetTestConfig()

	// Override with environment variables if present
	if token := os.Getenv("TEST_TELEGRAM_BOT_TOKEN"); token != "" {
		cfg.Bot.Token = token
	}

	if apiKey := os.Getenv("TEST_OPENWEATHER_API_KEY"); apiKey != "" {
		cfg.Weather.OpenWeatherAPIKey = apiKey
	}

	if dbPath := os.Getenv("TEST_DB_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}

	return cfg
}

// GetMinimalTestConfig returns bare minimum config for unit tests
func GetMinimalTestConfig() *config.Config {
	return &config.Config{
		Bot: config.BotConfig{
			Token: "test_token",
			Debug: true,
		},
		Weather: config.WeatherConfig{
			OpenWeatherAPIKey: "test_key",
			UserAgent:         "Test-Bot/1.0",
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "console",
		},
	}
}
