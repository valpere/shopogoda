package helpers

import (
	"os"

	"github.com/valpere/shopogoda/internal/config"
)

// GetTestConfig returns a configuration suitable for testing
func GetTestConfig() *config.Config {
	return &config.Config{
		Bot: config.BotConfig{
			Token: "test_bot_token",
			Debug: true,
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test_user",
			Password: "test_password",
			Name:     "test_db",
			SSLMode:  "disable",
		},
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
			DB:   1, // Use different DB for tests
		},
		Weather: config.WeatherConfig{
			OpenWeatherAPIKey: "test_weather_api_key",
			UserAgent:        "ShoPogoda-Weather-Bot/1.0 (test@shopogoda.bot)",
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
			GrafanaURL:     "http://localhost:3001",
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

	if dbHost := os.Getenv("TEST_DB_HOST"); dbHost != "" {
		cfg.Database.Host = dbHost
	}

	if redisHost := os.Getenv("TEST_REDIS_HOST"); redisHost != "" {
		cfg.Redis.Host = redisHost
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
			UserAgent:        "Test-Bot/1.0",
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "console",
		},
	}
}