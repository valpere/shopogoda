package main

import (
    "log"

    "github.com/valpere/enterprise-weather-bot/internal/config"
    "github.com/valpere/enterprise-weather-bot/internal/database"
    "github.com/valpere/enterprise-weather-bot/internal/models"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Connect to database
    db, err := database.Connect(&cfg.Database)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }

    // Run migrations
    if err := models.Migrate(db); err != nil {
        log.Fatalf("Failed to run migrations: %v", err)
    }

    log.Println("Migrations completed successfully!")
}
