package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/database"
	"github.com/valpere/shopogoda/internal/handlers/commands"
	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/internal/services"
)

var (
	botInstance      *gotgbot.Bot
	dispatcher       *ext.Dispatcher
	servicesInstance *services.Services
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
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	// Load configuration from environment
	cfg := &config.Config{
		Bot: config.BotConfig{
			Token:       os.Getenv("TELEGRAM_BOT_TOKEN"),
			Debug:       os.Getenv("BOT_DEBUG") == "true",
			WebhookMode: true,
			WebhookURL:  os.Getenv("BOT_WEBHOOK_URL"),
		},
		Database: config.DatabaseConfig{
			Host:     getEnv("DB_HOST", ""),
			Port:     getEnv("DB_PORT", "6543"),
			User:     getEnv("DB_USER", ""),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "postgres"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
		},
		Redis: config.RedisConfig{
			Host:     getEnv("REDIS_HOST", ""),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		Weather: config.WeatherConfig{
			APIKey:      os.Getenv("OPENWEATHER_API_KEY"),
			BaseURL:     "https://api.openweathermap.org/data/2.5",
			CacheTTL:    600 * time.Second,  // 10 minutes
			GeocodeTTL:  24 * time.Hour,     // 24 hours
			ForecastTTL: 3600 * time.Second, // 1 hour
		},
	}

	// Validate required fields
	if cfg.Bot.Token == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	if cfg.Weather.APIKey == "" {
		return fmt.Errorf("OPENWEATHER_API_KEY is required")
	}

	// Use DATABASE_URL if provided (Vercel style)
	var dsn string
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		dsn = dbURL
	} else {
		dsn = cfg.Database.DSN()
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
	if err := models.Migrate(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize Redis (use Upstash REST API if available)
	var redisClient *database.RedisClient
	if restURL := os.Getenv("UPSTASH_REDIS_REST_URL"); restURL != "" {
		// Use Upstash REST API (recommended for Vercel)
		log.Info().Msg("Using Upstash Redis REST API")
		// Note: Implement REST API client if needed
		// For now, Redis is optional - bot works without cache
		redisClient = nil
	} else if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		// Fallback to Redis protocol
		redisClient, err = database.NewRedisClientFromURL(redisURL)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to connect to Redis, continuing without cache")
			redisClient = nil
		}
	} else {
		// Try standard Redis config
		redisClient, err = database.NewRedisClient(cfg.Redis)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to connect to Redis, continuing without cache")
			redisClient = nil
		}
	}

	// Initialize services
	servicesInstance = services.New(db, redisClient, cfg, &log.Logger)

	// Create bot instance
	botInstance, err = gotgbot.NewBot(cfg.Bot.Token, &gotgbot.BotOpts{})
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	// Create dispatcher
	dispatcher = ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Error().Err(err).Msg("Update processing error")
			return ext.DispatcherActionNoop
		},
		MaxRoutines: 10, // Limit concurrent processing in serverless
	})

	// Register command handlers
	registerHandlers(dispatcher, servicesInstance)

	log.Info().Msg("Bot initialized successfully for Vercel serverless")
	return nil
}

func registerHandlers(dispatcher *ext.Dispatcher, services *services.Services) {
	// Command handlers
	dispatcher.AddHandler(commands.NewStartHandler(services))
	dispatcher.AddHandler(commands.NewWeatherHandler(services))
	dispatcher.AddHandler(commands.NewForecastHandler(services))
	dispatcher.AddHandler(commands.NewAirHandler(services))
	dispatcher.AddHandler(commands.NewSetLocationHandler(services))
	dispatcher.AddHandler(commands.NewSubscribeHandler(services))
	dispatcher.AddHandler(commands.NewAddAlertHandler(services))
	dispatcher.AddHandler(commands.NewSettingsHandler(services))
	dispatcher.AddHandler(commands.NewStatsHandler(services))
	dispatcher.AddHandler(commands.NewBroadcastHandler(services))
	dispatcher.AddHandler(commands.NewUsersHandler(services))

	// Callback query handlers
	dispatcher.AddHandler(commands.NewCallbackQueryHandler(services))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
