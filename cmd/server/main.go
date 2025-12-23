package main

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"imposter/internal/app"
	"imposter/internal/config"
	httpTransport "imposter/internal/transport/http"
)

//go:embed web/*
var webFS embed.FS

func main() {
	// Load configuration
	cfg := config.Load()

	// Set up logger
	var logger *slog.Logger
	logOpts := &slog.HandlerOptions{
		Level: parseLogLevel(cfg.Logging.Level),
	}

	if cfg.Logging.Format == "json" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, logOpts))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stdout, logOpts))
	}

	slog.SetDefault(logger)

	logger.Info("starting imposter game server",
		"env", cfg.Server.Env,
		"port", cfg.Server.Port,
	)

	// Create game hub
	hub := app.NewGameHub(logger)
	defer hub.Close()

	// Create HTTP server
	server := httpTransport.NewServer(cfg, hub, logger, webFS)

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("server stopped")
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
