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
	DemoMode    bool   `mapstructure:"demo_mode"`
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

	// Only search in current directory for Railway/production deployments
	// This prevents picking up cached config files from system directories
	viper.AddConfigPath(".")             // ./shopogoda.yaml (current directory)

	// Environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Map specific environment variables to config keys
	_ = viper.BindEnv("bot.token", "TELEGRAM_BOT_TOKEN")
	_ = viper.BindEnv("bot.debug", "BOT_DEBUG")
	_ = viper.BindEnv("bot.webhook_url", "BOT_WEBHOOK_URL")
	_ = viper.BindEnv("bot.webhook_port", "BOT_WEBHOOK_PORT")
	_ = viper.BindEnv("bot.demo_mode", "DEMO_MODE")

	_ = viper.BindEnv("database.host", "DB_HOST")
	_ = viper.BindEnv("database.port", "DB_PORT")
	_ = viper.BindEnv("database.user", "DB_USER")
	_ = viper.BindEnv("database.password", "DB_PASSWORD")
	_ = viper.BindEnv("database.name", "DB_NAME")
	_ = viper.BindEnv("database.ssl_mode", "DB_SSL_MODE")

	_ = viper.BindEnv("redis.host", "REDIS_HOST")
	_ = viper.BindEnv("redis.port", "REDIS_PORT")
	_ = viper.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = viper.BindEnv("redis.db", "REDIS_DB")

	_ = viper.BindEnv("weather.openweather_api_key", "OPENWEATHER_API_KEY")
	_ = viper.BindEnv("weather.airquality_api_key", "AIRQUALITY_API_KEY")
	_ = viper.BindEnv("weather.user_agent", "WEATHER_USER_AGENT")

	_ = viper.BindEnv("logging.level", "LOG_LEVEL")
	_ = viper.BindEnv("logging.format", "LOG_FORMAT")

	_ = viper.BindEnv("metrics.port", "PROMETHEUS_PORT")
	_ = viper.BindEnv("metrics.jaeger_endpoint", "JAEGER_ENDPOINT")

	_ = viper.BindEnv("integrations.slack_webhook_url", "SLACK_WEBHOOK_URL")
	_ = viper.BindEnv("integrations.teams_webhook_url", "TEAMS_WEBHOOK_URL")
	_ = viper.BindEnv("integrations.grafana_url", "GRAFANA_URL")

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
