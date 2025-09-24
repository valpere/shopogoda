package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Bot          BotConfig          `mapstructure:"bot"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Weather      WeatherConfig      `mapstructure:"weather"`
	Logging      LoggingConfig      `mapstructure:"logging"`
	Metrics      MetricsConfig      `mapstructure:"metrics"`
	Integrations IntegrationsConfig `mapstructure:"integrations"`
}

type BotConfig struct {
	Token       string `mapstructure:"token"`
	Debug       bool   `mapstructure:"debug"`
	WebhookURL  string `mapstructure:"webhook_url"`
	WebhookPort int    `mapstructure:"webhook_port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type WeatherConfig struct {
	OpenWeatherAPIKey string `mapstructure:"openweather_api_key"`
	AirQualityAPIKey  string `mapstructure:"airquality_api_key"`
	UserAgent         string `mapstructure:"user_agent"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type MetricsConfig struct {
	Port           int    `mapstructure:"port"`
	JaegerEndpoint string `mapstructure:"jaeger_endpoint"`
}

type IntegrationsConfig struct {
	SlackWebhookURL string `mapstructure:"slack_webhook_url"`
	TeamsWebhookURL string `mapstructure:"teams_webhook_url"`
	GrafanaURL      string `mapstructure:"grafana_url"`
}

func Load() (*Config, error) {
	// Load .env file if it exists
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	// Configure YAML config file search
	viper.SetConfigName("shopogoda")
	viper.SetConfigType("yaml")

	// Add search paths in order of precedence (first found wins)
	viper.AddConfigPath(".")             // ./shopogoda.yaml (current directory)
	viper.AddConfigPath("$HOME")         // ~/.shopogoda.yaml (home directory)
	viper.AddConfigPath("$HOME/.config") // ~/.config/shopogoda.yaml
	viper.AddConfigPath("/etc")          // /etc/shopogoda.yaml (system-wide)

	// Environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Map specific environment variables to config keys
	viper.BindEnv("bot.token", "TELEGRAM_BOT_TOKEN")
	viper.BindEnv("bot.debug", "BOT_DEBUG")
	viper.BindEnv("bot.webhook_url", "BOT_WEBHOOK_URL")
	viper.BindEnv("bot.webhook_port", "BOT_WEBHOOK_PORT")

	viper.BindEnv("database.host", "DB_HOST")
	viper.BindEnv("database.port", "DB_PORT")
	viper.BindEnv("database.user", "DB_USER")
	viper.BindEnv("database.password", "DB_PASSWORD")
	viper.BindEnv("database.name", "DB_NAME")
	viper.BindEnv("database.ssl_mode", "DB_SSL_MODE")

	viper.BindEnv("redis.host", "REDIS_HOST")
	viper.BindEnv("redis.port", "REDIS_PORT")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("redis.db", "REDIS_DB")

	viper.BindEnv("weather.openweather_api_key", "OPENWEATHER_API_KEY")
	viper.BindEnv("weather.airquality_api_key", "AIRQUALITY_API_KEY")
	viper.BindEnv("weather.user_agent", "WEATHER_USER_AGENT")

	viper.BindEnv("logging.level", "LOG_LEVEL")
	viper.BindEnv("logging.format", "LOG_FORMAT")

	viper.BindEnv("metrics.port", "PROMETHEUS_PORT")
	viper.BindEnv("metrics.jaeger_endpoint", "JAEGER_ENDPOINT")

	viper.BindEnv("integrations.slack_webhook_url", "SLACK_WEBHOOK_URL")
	viper.BindEnv("integrations.teams_webhook_url", "TEAMS_WEBHOOK_URL")
	viper.BindEnv("integrations.grafana_url", "GRAFANA_URL")

	// Set defaults
	setDefaults()

	// Read config file if exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	// Bot defaults
	viper.SetDefault("bot.debug", false)
	viper.SetDefault("bot.webhook_port", 8080)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.ssl_mode", "disable")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)

	// Weather defaults
	viper.SetDefault("weather.user_agent", "ShoPogoda-Weather-Bot/1.0 (contact@shopogoda.bot)")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")

	// Metrics defaults
	viper.SetDefault("metrics.port", 2112)
}
