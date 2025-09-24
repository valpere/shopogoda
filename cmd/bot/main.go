package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/valpere/shopogoda/internal/bot"
	"github.com/valpere/shopogoda/internal/config"
)

var Version = "dev"

func main() {
	log.Printf("Starting ShoPogoda v%s", Version)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize and start bot
	weatherBot, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Start bot in goroutine
	go func() {
		if err := weatherBot.Start(ctx); err != nil {
			log.Fatalf("Failed to start bot: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down ShoPogoda...")
	cancel()

	if err := weatherBot.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("ShoPogoda stopped gracefully")
}
