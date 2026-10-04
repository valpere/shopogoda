package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Reset viper state before each test
	resetViper := func() {
		viper.Reset()
	}

	t.Run("loads with defaults when no config file exists", func(t *testing.T) {
		resetViper()

		// Ensure we're in a directory without config files
		originalDir, _ := os.Getwd()
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Verify defaults are applied
		assert.Equal(t, false, cfg.Bot.Debug)
		assert.Equal(t, 8080, cfg.Bot.WebhookPort)
		assert.Equal(t, "./data/shopogoda.db", cfg.Database.Path)
		assert.Equal(t, "info", cfg.Logging.Level)
		assert.Equal(t, "json", cfg.Logging.Format)
		assert.Equal(t, 2112, cfg.Metrics.Port)
	})

	t.Run("loads from environment variables", func(t *testing.T) {
		resetViper()

		// Set environment variables
		require.NoError(t, os.Setenv("TELEGRAM_BOT_TOKEN", "test_token_123"))
		require.NoError(t, os.Setenv("BOT_DEBUG", "true"))
		require.NoError(t, os.Setenv("DB_PATH", "/var/lib/shopogoda/test.db"))
		require.NoError(t, os.Setenv("OPENWEATHER_API_KEY", "weather_key_123"))
		require.NoError(t, os.Setenv("LOG_LEVEL", "debug"))
		require.NoError(t, os.Setenv("PROMETHEUS_PORT", "9090"))
		defer func() {
			_ = os.Unsetenv("TELEGRAM_BOT_TOKEN")
			_ = os.Unsetenv("BOT_DEBUG")
			_ = os.Unsetenv("DB_PATH")
			_ = os.Unsetenv("OPENWEATHER_API_KEY")
			_ = os.Unsetenv("LOG_LEVEL")
			_ = os.Unsetenv("PROMETHEUS_PORT")
		}()

		// Ensure we're in a directory without config files
		originalDir, _ := os.Getwd()
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Verify environment variables are loaded
		assert.Equal(t, "test_token_123", cfg.Bot.Token)
		assert.Equal(t, true, cfg.Bot.Debug)
		assert.Equal(t, "/var/lib/shopogoda/test.db", cfg.Database.Path)
		assert.Equal(t, "weather_key_123", cfg.Weather.OpenWeatherAPIKey)
		assert.Equal(t, "debug", cfg.Logging.Level)
		assert.Equal(t, 9090, cfg.Metrics.Port)
	})

	t.Run("handles missing .env file gracefully", func(t *testing.T) {
		resetViper()

		// Ensure we're in a directory without .env file
		originalDir, _ := os.Getwd()
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("handles invalid .env file", func(t *testing.T) {
		resetViper()

		// Create a temp directory with an invalid .env file
		originalDir, _ := os.Getwd()
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		// Create .env file with invalid format (unclosed quote)
		err := os.WriteFile(".env", []byte("INVALID_VAR=\"unclosed quote\n"), 0644)
		require.NoError(t, err)

		cfg, err := Load()
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "error loading .env file")
	})

	t.Run("handles invalid config file format", func(t *testing.T) {
		resetViper()

		// Create a temp directory with an invalid YAML config file
		originalDir, _ := os.Getwd()
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		// Create shopogoda.yaml with invalid YAML syntax
		invalidYAML := `
bot:
  token: "test"
  debug: [invalid - unclosed bracket
`
		err := os.WriteFile("shopogoda.yaml", []byte(invalidYAML), 0644)
		require.NoError(t, err)

		// YAML loading is disabled for Railway compatibility,
		// so invalid YAML files are ignored and config loads from env vars/defaults
		cfg, err := Load()
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, 8080, cfg.Bot.WebhookPort) // Should have default values
	})
}

func TestSetDefaults(t *testing.T) {
	// Reset viper state
	viper.Reset()

	setDefaults()

	t.Run("bot defaults", func(t *testing.T) {
		assert.Equal(t, false, viper.GetBool("bot.debug"))
		assert.Equal(t, 8080, viper.GetInt("bot.webhook_port"))
	})

	t.Run("database defaults", func(t *testing.T) {
		assert.Equal(t, "./data/shopogoda.db", viper.GetString("database.path"))
	})

	t.Run("weather defaults", func(t *testing.T) {
		assert.Equal(t, "ShoPogoda-Weather-Bot/1.0 (contact@shopogoda.bot)",
			viper.GetString("weather.user_agent"))
	})

	t.Run("logging defaults", func(t *testing.T) {
		assert.Equal(t, "info", viper.GetString("logging.level"))
		assert.Equal(t, "json", viper.GetString("logging.format"))
	})

	t.Run("metrics defaults", func(t *testing.T) {
		assert.Equal(t, 2112, viper.GetInt("metrics.port"))
	})
}

func TestBotConfig(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		cfg := BotConfig{
			Token:       "bot_token",
			Debug:       true,
			WebhookURL:  "https://example.com/webhook",
			WebhookPort: 8443,
		}

		assert.Equal(t, "bot_token", cfg.Token)
		assert.Equal(t, true, cfg.Debug)
		assert.Equal(t, "https://example.com/webhook", cfg.WebhookURL)
		assert.Equal(t, 8443, cfg.WebhookPort)
	})

	t.Run("zero values", func(t *testing.T) {
		cfg := BotConfig{}

		assert.Equal(t, "", cfg.Token)
		assert.Equal(t, false, cfg.Debug)
		assert.Equal(t, "", cfg.WebhookURL)
		assert.Equal(t, 0, cfg.WebhookPort)
	})
}

func TestDatabaseConfig(t *testing.T) {
	cfg := DatabaseConfig{Path: "/data/shopogoda.db"}
	assert.Equal(t, "/data/shopogoda.db", cfg.Path)
}

func TestWeatherConfig(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		cfg := WeatherConfig{
			OpenWeatherAPIKey: "openweather_key",
			AirQualityAPIKey:  "airquality_key",
			UserAgent:         "TestBot/1.0",
		}

		assert.Equal(t, "openweather_key", cfg.OpenWeatherAPIKey)
		assert.Equal(t, "airquality_key", cfg.AirQualityAPIKey)
		assert.Equal(t, "TestBot/1.0", cfg.UserAgent)
	})

	t.Run("with only required fields", func(t *testing.T) {
		cfg := WeatherConfig{
			OpenWeatherAPIKey: "key",
			UserAgent:         "Bot/1.0",
		}

		assert.Equal(t, "key", cfg.OpenWeatherAPIKey)
		assert.Equal(t, "", cfg.AirQualityAPIKey)
		assert.Equal(t, "Bot/1.0", cfg.UserAgent)
	})
}

func TestLoggingConfig(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		cfg := LoggingConfig{
			Level:  "debug",
			Format: "json",
		}

		assert.Equal(t, "debug", cfg.Level)
		assert.Equal(t, "json", cfg.Format)
	})

	t.Run("log levels", func(t *testing.T) {
		levels := []string{"debug", "info", "warn", "error", "fatal"}

		for _, level := range levels {
			cfg := LoggingConfig{Level: level}
			assert.Equal(t, level, cfg.Level)
		}
	})

	t.Run("log formats", func(t *testing.T) {
		formats := []string{"json", "console", "text"}

		for _, format := range formats {
			cfg := LoggingConfig{Format: format}
			assert.Equal(t, format, cfg.Format)
		}
	})
}

func TestMetricsConfig(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		cfg := MetricsConfig{
			Port:           9090,
			JaegerEndpoint: "http://jaeger:14268/api/traces",
		}

		assert.Equal(t, 9090, cfg.Port)
		assert.Equal(t, "http://jaeger:14268/api/traces", cfg.JaegerEndpoint)
	})

	t.Run("without jaeger", func(t *testing.T) {
		cfg := MetricsConfig{
			Port: 2112,
		}

		assert.Equal(t, 2112, cfg.Port)
		assert.Equal(t, "", cfg.JaegerEndpoint)
	})
}

func TestIntegrationsConfig(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		cfg := IntegrationsConfig{
			SlackWebhookURL: "https://hooks.slack.com/services/XXX",
			TeamsWebhookURL: "https://outlook.office.com/webhook/XXX",
			GrafanaURL:      "http://grafana:3000",
		}

		assert.Equal(t, "https://hooks.slack.com/services/XXX", cfg.SlackWebhookURL)
		assert.Equal(t, "https://outlook.office.com/webhook/XXX", cfg.TeamsWebhookURL)
		assert.Equal(t, "http://grafana:3000", cfg.GrafanaURL)
	})

	t.Run("with only Slack", func(t *testing.T) {
		cfg := IntegrationsConfig{
			SlackWebhookURL: "https://hooks.slack.com/services/XXX",
		}

		assert.Equal(t, "https://hooks.slack.com/services/XXX", cfg.SlackWebhookURL)
		assert.Equal(t, "", cfg.TeamsWebhookURL)
		assert.Equal(t, "", cfg.GrafanaURL)
	})

	t.Run("empty integrations", func(t *testing.T) {
		cfg := IntegrationsConfig{}

		assert.Equal(t, "", cfg.SlackWebhookURL)
		assert.Equal(t, "", cfg.TeamsWebhookURL)
		assert.Equal(t, "", cfg.GrafanaURL)
	})
}

func TestConfig(t *testing.T) {
	t.Run("full config structure", func(t *testing.T) {
		cfg := Config{
			Bot: BotConfig{
				Token: "token",
				Debug: true,
			},
			Database: DatabaseConfig{
				Path: "./data/shopogoda.db",
			},
			Weather: WeatherConfig{
				OpenWeatherAPIKey: "key",
			},
			Logging: LoggingConfig{
				Level:  "info",
				Format: "json",
			},
			Metrics: MetricsConfig{
				Port: 2112,
			},
			Integrations: IntegrationsConfig{
				SlackWebhookURL: "https://slack.example.com",
			},
		}

		assert.NotNil(t, cfg.Bot)
		assert.NotNil(t, cfg.Database)
		assert.NotNil(t, cfg.Weather)
		assert.NotNil(t, cfg.Logging)
		assert.NotNil(t, cfg.Metrics)
		assert.NotNil(t, cfg.Integrations)

		assert.Equal(t, "token", cfg.Bot.Token)
		assert.Equal(t, "./data/shopogoda.db", cfg.Database.Path)
		assert.Equal(t, "key", cfg.Weather.OpenWeatherAPIKey)
		assert.Equal(t, "info", cfg.Logging.Level)
		assert.Equal(t, 2112, cfg.Metrics.Port)
		assert.Equal(t, "https://slack.example.com", cfg.Integrations.SlackWebhookURL)
	})
}

func TestEnvironmentVariableMapping(t *testing.T) {
	t.Run("all environment variables are mapped", func(t *testing.T) {
		viper.Reset()

		// Set all environment variables
		envVars := map[string]string{
			"TELEGRAM_BOT_TOKEN":  "bot_token",
			"BOT_DEBUG":           "true",
			"BOT_WEBHOOK_URL":     "https://example.com",
			"BOT_WEBHOOK_PORT":    "8443",
			"DB_PATH":             "/data/shopogoda.db",
			"OPENWEATHER_API_KEY": "weather_key",
			"AIRQUALITY_API_KEY":  "air_key",
			"WEATHER_USER_AGENT":  "TestBot/1.0",
			"LOG_LEVEL":           "debug",
			"LOG_FORMAT":          "console",
			"PROMETHEUS_PORT":     "9090",
			"JAEGER_ENDPOINT":     "http://jaeger:14268",
			"SLACK_WEBHOOK_URL":   "https://slack.example.com",
			"TEAMS_WEBHOOK_URL":   "https://teams.example.com",
			"GRAFANA_URL":         "http://grafana:3000",
		}

		for key, value := range envVars {
			require.NoError(t, os.Setenv(key, value))
		}
		defer func() {
			for key := range envVars {
				_ = os.Unsetenv(key)
			}
		}()

		// Load config
		originalDir, _ := os.Getwd()
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		cfg, err := Load()
		require.NoError(t, err)

		// Verify all values are loaded
		assert.Equal(t, "bot_token", cfg.Bot.Token)
		assert.Equal(t, true, cfg.Bot.Debug)
		assert.Equal(t, "https://example.com", cfg.Bot.WebhookURL)
		assert.Equal(t, 8443, cfg.Bot.WebhookPort)
		assert.Equal(t, "/data/shopogoda.db", cfg.Database.Path)
		assert.Equal(t, "weather_key", cfg.Weather.OpenWeatherAPIKey)
		assert.Equal(t, "air_key", cfg.Weather.AirQualityAPIKey)
		assert.Equal(t, "TestBot/1.0", cfg.Weather.UserAgent)
		assert.Equal(t, "debug", cfg.Logging.Level)
		assert.Equal(t, "console", cfg.Logging.Format)
		assert.Equal(t, 9090, cfg.Metrics.Port)
		assert.Equal(t, "http://jaeger:14268", cfg.Metrics.JaegerEndpoint)
		assert.Equal(t, "https://slack.example.com", cfg.Integrations.SlackWebhookURL)
		assert.Equal(t, "https://teams.example.com", cfg.Integrations.TeamsWebhookURL)
		assert.Equal(t, "http://grafana:3000", cfg.Integrations.GrafanaURL)
	})
}
