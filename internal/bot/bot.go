package bot

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "strconv"
    "time"

    "github.com/PaulSonOfLars/gotgbot/v2"
    "github.com/PaulSonOfLars/gotgbot/v2/ext"
    "github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
    "github.com/gin-gonic/gin"
    "github.com/rs/zerolog"

    "github.com/valpere/shopogoda/internal/config"
    "github.com/valpere/shopogoda/internal/database"
    "github.com/valpere/shopogoda/internal/handlers/commands"
    // "github.com/valpere/shopogoda/internal/middleware" // Not used yet
    "github.com/valpere/shopogoda/internal/services"
    "github.com/valpere/shopogoda/pkg/metrics"
)

type Bot struct {
    bot        *gotgbot.Bot
    updater    *ext.Updater
    dispatcher *ext.Dispatcher
    config     *config.Config
    logger     zerolog.Logger
    services   *services.Services
    server     *http.Server
    metrics    *metrics.Metrics
}

func New(cfg *config.Config) (*Bot, error) {
    // Initialize logger
    logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
        With().
        Timestamp().
        Str("component", "bot").
        Logger()

    // Initialize metrics
    metricsCollector := metrics.New()

    // Initialize database
    db, err := database.Connect(&cfg.Database)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // Initialize Redis
    rdb, err := database.ConnectRedis(&cfg.Redis)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to Redis: %w", err)
    }

    // Initialize services
    services := services.New(db, rdb, cfg, &logger)

    // Create bot
    botInstance, err := gotgbot.NewBot(cfg.Bot.Token, &gotgbot.BotOpts{
        BotClient: &gotgbot.BaseBotClient{
            Client: http.Client{Timeout: 30 * time.Second},
        },
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create bot: %w", err)
    }

    // Create updater and dispatcher
    dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{})
    updater := ext.NewUpdater(dispatcher, &ext.UpdaterOpts{})

    // Create bot instance
    weatherBot := &Bot{
        bot:        botInstance,
        updater:    updater,
        dispatcher: dispatcher,
        config:     cfg,
        logger:     logger,
        services:   services,
        metrics:    metricsCollector,
    }

    // Setup handlers
    if err := weatherBot.setupHandlers(); err != nil {
        return nil, fmt.Errorf("failed to setup handlers: %w", err)
    }

    // Setup HTTP server for webhooks and metrics
    weatherBot.setupHTTPServer()

    return weatherBot, nil
}

func (b *Bot) setupHandlers() error {
    // Add middleware (commented out for now due to implementation issues)
    // b.dispatcher.AddHandlerToGroup(middleware.Logging(b.logger), -1)
    // b.dispatcher.AddHandlerToGroup(middleware.Metrics(), -1)
    // b.dispatcher.AddHandlerToGroup(middleware.UserRegistration(b.services), -1)
    // b.dispatcher.AddHandlerToGroup(middleware.RateLimiting(), -1)

    // Command handlers
    cmdHandler := commands.New(b.services, &b.logger)

    // Basic commands
    b.dispatcher.AddHandler(handlers.NewCommand("start", cmdHandler.Start))
    b.dispatcher.AddHandler(handlers.NewCommand("help", cmdHandler.Help))
    b.dispatcher.AddHandler(handlers.NewCommand("settings", cmdHandler.Settings))

    // Weather commands
    b.dispatcher.AddHandler(handlers.NewCommand("weather", cmdHandler.CurrentWeather))
    b.dispatcher.AddHandler(handlers.NewCommand("forecast", cmdHandler.Forecast))
    b.dispatcher.AddHandler(handlers.NewCommand("air", cmdHandler.AirQuality))

    // Location management
    b.dispatcher.AddHandler(handlers.NewCommand("addlocation", cmdHandler.AddLocation))
    b.dispatcher.AddHandler(handlers.NewCommand("locations", cmdHandler.ListLocations))
    b.dispatcher.AddHandler(handlers.NewCommand("setdefault", cmdHandler.SetDefaultLocation))

    // Subscription management
    b.dispatcher.AddHandler(handlers.NewCommand("subscribe", cmdHandler.Subscribe))
    b.dispatcher.AddHandler(handlers.NewCommand("unsubscribe", cmdHandler.Unsubscribe))
    b.dispatcher.AddHandler(handlers.NewCommand("subscriptions", cmdHandler.ListSubscriptions))

    // Alert management
    b.dispatcher.AddHandler(handlers.NewCommand("addalert", cmdHandler.AddAlert))
    b.dispatcher.AddHandler(handlers.NewCommand("alerts", cmdHandler.ListAlerts))
    b.dispatcher.AddHandler(handlers.NewCommand("removealert", cmdHandler.RemoveAlert))

    // Admin commands (role-based access)
    b.dispatcher.AddHandler(handlers.NewCommand("stats", cmdHandler.AdminStats))
    b.dispatcher.AddHandler(handlers.NewCommand("broadcast", cmdHandler.AdminBroadcast))
    b.dispatcher.AddHandler(handlers.NewCommand("users", cmdHandler.AdminListUsers))

    // Callback query handlers
    b.dispatcher.AddHandler(handlers.NewCallback(nil, cmdHandler.HandleCallback))

    // Message handlers for location sharing
    b.dispatcher.AddHandler(handlers.NewMessage(nil, cmdHandler.HandleLocationMessage))

    return nil
}

func (b *Bot) setupHTTPServer() {
    gin.SetMode(gin.ReleaseMode)
    router := gin.New()
    router.Use(gin.Recovery())

    // Health check endpoint
    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status":  "healthy",
            "version": "1.0.0",
            "time":    time.Now().Unix(),
        })
    })

    // Metrics endpoint
    router.GET("/metrics", gin.WrapH(b.metrics.Handler()))

    // Webhook endpoint
    if b.config.Bot.WebhookURL != "" {
        router.POST("/webhook", func(c *gin.Context) {
            var update gotgbot.Update
            if err := c.ShouldBindJSON(&update); err != nil {
                b.logger.Error().Err(err).Msg("Failed to parse webhook update")
                c.Status(http.StatusBadRequest)
                return
            }

            if err := b.dispatcher.ProcessUpdate(b.bot, &update, nil); err != nil {
                b.logger.Error().Err(err).Msg("Failed to process update")
                c.Status(http.StatusInternalServerError)
                return
            }

            c.Status(http.StatusOK)
        })
    }

    b.server = &http.Server{
        Addr:         ":" + strconv.Itoa(b.config.Bot.WebhookPort),
        Handler:      router,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
    }
}

func (b *Bot) Start(ctx context.Context) error {
    b.logger.Info().Msg("Starting ShoPogoda bot...")

    // Start HTTP server
    go func() {
        if err := b.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            b.logger.Fatal().Err(err).Msg("HTTP server failed to start")
        }
    }()

    b.logger.Info().
        Int("port", b.config.Bot.WebhookPort).
        Msg("HTTP server started")

    // Setup webhook or polling
    if b.config.Bot.WebhookURL != "" {
        if err := b.setupWebhook(); err != nil {
            return fmt.Errorf("failed to setup webhook: %w", err)
        }
    } else {
        b.logger.Info().Msg("Starting polling...")
        if err := b.updater.StartPolling(b.bot, &ext.PollingOpts{
            DropPendingUpdates: true,
            GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
                Timeout: 10,
                RequestOpts: &gotgbot.RequestOpts{
                    Timeout: time.Second * 15,
                },
            },
        }); err != nil {
            return fmt.Errorf("failed to start polling: %w", err)
        }
    }

    // Start background services
    go b.services.StartScheduler(ctx)

    b.logger.Info().Msg("ShoPogoda bot started successfully")

    // Wait for context cancellation
    <-ctx.Done()
    return nil
}

func (b *Bot) setupWebhook() error {
    webhookURL := b.config.Bot.WebhookURL + "/webhook"

    _, err := b.bot.SetWebhook(webhookURL, &gotgbot.SetWebhookOpts{
        MaxConnections:     100,
        DropPendingUpdates: true,
    })
    if err != nil {
        return fmt.Errorf("failed to set webhook: %w", err)
    }

    b.logger.Info().
        Str("webhook_url", webhookURL).
        Msg("Webhook configured")

    return nil
}

func (b *Bot) Stop() error {
    b.logger.Info().Msg("Stopping ShoPogoda bot...")

    // Stop updater
    b.updater.Stop()

    // Shutdown HTTP server
    if b.server != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := b.server.Shutdown(ctx); err != nil {
            b.logger.Error().Err(err).Msg("HTTP server shutdown error")
        }
    }

    // Stop services
    b.services.Stop()

    b.logger.Info().Msg("ShoPogoda bot stopped")
    return nil
}
