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
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, 5432, cfg.Database.Port)
		assert.Equal(t, "disable", cfg.Database.SSLMode)
		assert.Equal(t, "localhost", cfg.Redis.Host)
		assert.Equal(t, 6379, cfg.Redis.Port)
		assert.Equal(t, 0, cfg.Redis.DB)
		assert.Equal(t, "info", cfg.Logging.Level)
		assert.Equal(t, "json", cfg.Logging.Format)
		assert.Equal(t, 2112, cfg.Metrics.Port)
	})

	t.Run("loads from environment variables", func(t *testing.T) {
		resetViper()

		// Set environment variables
		os.Setenv("TELEGRAM_BOT_TOKEN", "test_token_123")
		os.Setenv("BOT_DEBUG", "true")
		os.Setenv("DB_HOST", "postgres.example.com")
		os.Setenv("DB_PORT", "5433")
		os.Setenv("REDIS_HOST", "redis.example.com")
		os.Setenv("REDIS_PORT", "6380")
		os.Setenv("OPENWEATHER_API_KEY", "weather_key_123")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("PROMETHEUS_PORT", "9090")
		defer func() {
			os.Unsetenv("TELEGRAM_BOT_TOKEN")
			os.Unsetenv("BOT_DEBUG")
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
			os.Unsetenv("REDIS_HOST")
			os.Unsetenv("REDIS_PORT")
			os.Unsetenv("OPENWEATHER_API_KEY")
			os.Unsetenv("LOG_LEVEL")
			os.Unsetenv("PROMETHEUS_PORT")
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
		assert.Equal(t, "postgres.example.com", cfg.Database.Host)
		assert.Equal(t, 5433, cfg.Database.Port)
		assert.Equal(t, "redis.example.com", cfg.Redis.Host)
		assert.Equal(t, 6380, cfg.Redis.Port)
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
		assert.Equal(t, "localhost", viper.GetString("database.host"))
		assert.Equal(t, 5432, viper.GetInt("database.port"))
		assert.Equal(t, "disable", viper.GetString("database.ssl_mode"))
	})

	t.Run("redis defaults", func(t *testing.T) {
		assert.Equal(t, "localhost", viper.GetString("redis.host"))
		assert.Equal(t, 6379, viper.GetInt("redis.port"))
		assert.Equal(t, 0, viper.GetInt("redis.db"))
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
	t.Run("all fields", func(t *testing.T) {
		cfg := DatabaseConfig{
			Host:     "db.example.com",
			Port:     5432,
			User:     "admin",
			Password: "secret",
			Name:     "shopogoda",
			SSLMode:  "require",
		}

		assert.Equal(t, "db.example.com", cfg.Host)
		assert.Equal(t, 5432, cfg.Port)
		assert.Equal(t, "admin", cfg.User)
		assert.Equal(t, "secret", cfg.Password)
		assert.Equal(t, "shopogoda", cfg.Name)
		assert.Equal(t, "require", cfg.SSLMode)
	})

	t.Run("SSL modes", func(t *testing.T) {
		modes := []string{"disable", "require", "verify-ca", "verify-full"}

		for _, mode := range modes {
			cfg := DatabaseConfig{SSLMode: mode}
			assert.Equal(t, mode, cfg.SSLMode)
		}
	})
}

func TestRedisConfig(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		cfg := RedisConfig{
			Host:     "redis.example.com",
			Port:     6380,
			Password: "redis_pass",
			DB:       5,
		}

		assert.Equal(t, "redis.example.com", cfg.Host)
		assert.Equal(t, 6380, cfg.Port)
		assert.Equal(t, "redis_pass", cfg.Password)
		assert.Equal(t, 5, cfg.DB)
	})

	t.Run("database numbers", func(t *testing.T) {
		// Redis supports DB 0-15
		for db := 0; db <= 15; db++ {
			cfg := RedisConfig{DB: db}
			assert.Equal(t, db, cfg.DB)
		}
	})
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
				Host: "localhost",
				Port: 5432,
			},
			Redis: RedisConfig{
				Host: "localhost",
				Port: 6379,
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
		assert.NotNil(t, cfg.Redis)
		assert.NotNil(t, cfg.Weather)
		assert.NotNil(t, cfg.Logging)
		assert.NotNil(t, cfg.Metrics)
		assert.NotNil(t, cfg.Integrations)

		assert.Equal(t, "token", cfg.Bot.Token)
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, "localhost", cfg.Redis.Host)
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
			"TELEGRAM_BOT_TOKEN":     "bot_token",
			"BOT_DEBUG":              "true",
			"BOT_WEBHOOK_URL":        "https://example.com",
			"BOT_WEBHOOK_PORT":       "8443",
			"DB_HOST":                "db.example.com",
			"DB_PORT":                "5433",
			"DB_USER":                "admin",
			"DB_PASSWORD":            "secret",
			"DB_NAME":                "shopogoda",
			"DB_SSL_MODE":            "require",
			"REDIS_HOST":             "redis.example.com",
			"REDIS_PORT":             "6380",
			"REDIS_PASSWORD":         "redis_pass",
			"REDIS_DB":               "5",
			"OPENWEATHER_API_KEY":    "weather_key",
			"AIRQUALITY_API_KEY":     "air_key",
			"WEATHER_USER_AGENT":     "TestBot/1.0",
			"LOG_LEVEL":              "debug",
			"LOG_FORMAT":             "console",
			"PROMETHEUS_PORT":        "9090",
			"JAEGER_ENDPOINT":        "http://jaeger:14268",
			"SLACK_WEBHOOK_URL":      "https://slack.example.com",
			"TEAMS_WEBHOOK_URL":      "https://teams.example.com",
			"GRAFANA_URL":            "http://grafana:3000",
		}

		for key, value := range envVars {
			os.Setenv(key, value)
		}
		defer func() {
			for key := range envVars {
				os.Unsetenv(key)
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
		assert.Equal(t, "db.example.com", cfg.Database.Host)
		assert.Equal(t, 5433, cfg.Database.Port)
		assert.Equal(t, "admin", cfg.Database.User)
		assert.Equal(t, "secret", cfg.Database.Password)
		assert.Equal(t, "shopogoda", cfg.Database.Name)
		assert.Equal(t, "require", cfg.Database.SSLMode)
		assert.Equal(t, "redis.example.com", cfg.Redis.Host)
		assert.Equal(t, 6380, cfg.Redis.Port)
		assert.Equal(t, "redis_pass", cfg.Redis.Password)
		assert.Equal(t, 5, cfg.Redis.DB)
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
