package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/valpere/shopogoda/internal/bot"
	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/version"
)

func main() {
	// Command-line flags
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	// Handle version flag
	if *versionFlag {
		info := version.GetInfo()
		fmt.Println(info.String())
		os.Exit(0)
	}

	log.Printf("Starting ShoPogoda v%s", version.Version)

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
