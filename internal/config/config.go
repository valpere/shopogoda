package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Bot      BotConfig      `mapstructure:"bot"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Weather  WeatherConfig  `mapstructure:"weather"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Integrations IntegrationsConfig `mapstructure:"integrations"`
}

type BotConfig struct {
	Token      string `mapstructure:"token"`
	Debug      bool   `mapstructure:"debug"`
	WebhookURL string `mapstructure:"webhook_url"`
	WebhookPort int   `mapstructure:"webhook_port"`
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
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	// Environment variables
	viper.SetEnvPrefix("WB") // Weather Bot
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

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

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")

	// Metrics defaults
	viper.SetDefault("metrics.port", 2112)
}
