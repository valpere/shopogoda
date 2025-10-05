package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/database"
	"github.com/valpere/shopogoda/internal/handlers/commands"
	"github.com/valpere/shopogoda/internal/services"
)

var (
	botInstance      *gotgbot.Bot
	dispatcher       *ext.Dispatcher
	servicesInstance *services.Services
	logger           zerolog.Logger
	initOnce         sync.Once
	initErr          error
)

// Handler is the Vercel serverless function entry point for Telegram webhooks
func Handler(w http.ResponseWriter, r *http.Request) {
	// Initialize on first request (cold start)
	initOnce.Do(func() {
		initErr = initialize()
	})

	if initErr != nil {
		log.Error().Err(initErr).Msg("Bot initialization failed")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Only accept POST requests to /webhook
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read request body")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse Telegram update
	var update gotgbot.Update
	if err := json.Unmarshal(body, &update); err != nil {
		log.Error().Err(err).Msg("Failed to parse update")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Process update with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Second) // Vercel has 10s limit
	defer cancel()

	// Process in background to avoid blocking response
	go func() {
		if err := dispatcher.ProcessUpdate(botInstance, &update, map[string]any{"ctx": ctx}); err != nil {
			log.Error().Err(err).Msg("Failed to process update")
		}
	}()

	// Always return 200 OK to Telegram immediately
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func initialize() error {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("LOG_FORMAT") == "json" {
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	// Load configuration from environment
	cfg := &config.Config{
		Bot: config.BotConfig{
			Token:       os.Getenv("TELEGRAM_BOT_TOKEN"),
			Debug:       os.Getenv("BOT_DEBUG") == "true",
			WebhookURL:  os.Getenv("BOT_WEBHOOK_URL"),
			WebhookPort: 8080,
		},
		Database: config.DatabaseConfig{
			Host:     getEnv("DB_HOST", ""),
			Port:     getEnvInt("DB_PORT", 6543),
			User:     getEnv("DB_USER", ""),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "postgres"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
		},
		Redis: config.RedisConfig{
			Host:     getEnv("REDIS_HOST", ""),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		Weather: config.WeatherConfig{
			OpenWeatherAPIKey: os.Getenv("OPENWEATHER_API_KEY"),
			AirQualityAPIKey:  os.Getenv("AIRQUALITY_API_KEY"),
			UserAgent:         getEnv("WEATHER_USER_AGENT", "ShoPogoda-Weather-Bot/1.0"),
		},
	}

	// Validate required fields
	if cfg.Bot.Token == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	if cfg.Weather.OpenWeatherAPIKey == "" {
		return fmt.Errorf("OPENWEATHER_API_KEY is required")
	}

	// Use DATABASE_URL if provided (Vercel style)
	var dsn string
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		dsn = dbURL
	} else {
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
			cfg.Database.Password, cfg.Database.Name, cfg.Database.SSLMode)
	}

	// Initialize database with connection pooling settings for serverless
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool for serverless (small pool, short lifetime)
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	sqlDB.SetMaxOpenConns(5)                  // Small pool for serverless
	sqlDB.SetMaxIdleConns(2)                  // Minimal idle connections
	sqlDB.SetConnMaxLifetime(5 * time.Minute) // Short lifetime

	// Run migrations
	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize Redis (optional - bot works without cache)
	var redisClient *redis.Client
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		// Note: For Vercel, consider using Upstash REST API instead
		// This is a fallback for standard Redis protocol
		logger.Warn().Msg("Redis connection not implemented for serverless - running without cache")
		redisClient = nil
	}

	// Initialize services
	servicesInstance = services.New(db, redisClient, cfg, &logger)

	// Create bot instance
	botInstance, err = gotgbot.NewBot(cfg.Bot.Token, &gotgbot.BotOpts{
		BotClient: &gotgbot.BaseBotClient{
			Client: http.Client{Timeout: 30 * time.Second},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	// Set bot instance for notifications
	servicesInstance.Notification.SetBot(botInstance)

	// Create dispatcher
	dispatcher = ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			logger.Error().Err(err).Msg("Update processing error")
			return ext.DispatcherActionNoop
		},
		MaxRoutines: 10, // Limit concurrent processing in serverless
	})

	// Register command handlers
	setupHandlers(dispatcher, servicesInstance)

	logger.Info().Msg("Bot initialized successfully for Vercel serverless")
	return nil
}

func setupHandlers(d *ext.Dispatcher, svc *services.Services) {
	// Create command handler instance
	cmdHandler := commands.New(svc, &logger)

	// Basic commands
	d.AddHandler(handlers.NewCommand("start", cmdHandler.Start))
	d.AddHandler(handlers.NewCommand("help", cmdHandler.Help))
	d.AddHandler(handlers.NewCommand("settings", cmdHandler.Settings))
	d.AddHandler(handlers.NewCommand("language", cmdHandler.Language))
	d.AddHandler(handlers.NewCommand("version", cmdHandler.Version))

	// Weather commands
	d.AddHandler(handlers.NewCommand("weather", cmdHandler.CurrentWeather))
	d.AddHandler(handlers.NewCommand("forecast", cmdHandler.Forecast))
	d.AddHandler(handlers.NewCommand("air", cmdHandler.AirQuality))

	// Location management
	d.AddHandler(handlers.NewCommand("setlocation", cmdHandler.SetLocation))

	// Subscription management
	d.AddHandler(handlers.NewCommand("subscribe", cmdHandler.Subscribe))
	d.AddHandler(handlers.NewCommand("unsubscribe", cmdHandler.Unsubscribe))
	d.AddHandler(handlers.NewCommand("subscriptions", cmdHandler.ListSubscriptions))

	// Alert management
	d.AddHandler(handlers.NewCommand("addalert", cmdHandler.AddAlert))
	d.AddHandler(handlers.NewCommand("alerts", cmdHandler.ListAlerts))
	d.AddHandler(handlers.NewCommand("removealert", cmdHandler.RemoveAlert))

	// Admin commands
	d.AddHandler(handlers.NewCommand("stats", cmdHandler.AdminStats))
	d.AddHandler(handlers.NewCommand("broadcast", cmdHandler.AdminBroadcast))
	d.AddHandler(handlers.NewCommand("users", cmdHandler.AdminListUsers))

	// Callback query handlers
	d.AddHandler(handlers.NewCallback(nil, cmdHandler.HandleCallback))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
