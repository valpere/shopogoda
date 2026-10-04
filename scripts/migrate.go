package main

import (
	"log"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/database"
	"github.com/valpere/shopogoda/internal/models"
)

func main() {
	log.Println("Starting ShoPogoda database migrations...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database (creates the SQLite file if missing)
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Running AutoMigrate...")
	if err := models.Migrate(db); err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}

	log.Println("ShoPogoda migrations completed successfully!")
}
