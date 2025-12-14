package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"constructor-script-backend/internal/app"
	"constructor-script-backend/internal/config"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/validator"
)

func main() {
	logger.Init()
	// Ensure any log file opened by the logger is closed on exit
	defer func() {
		if err := logger.Close(); err != nil {
			logger.Error(err, "Failed to close log file", nil)
		}
	}()
	logger.Info("Starting Constructor Script CMS", nil)

	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, using environment variables", nil)
	}

	cfg := config.New()
	validator.Init()

	application, err := app.New(cfg, app.Options{ThemesDir: "./themes", DefaultTheme: "default", PluginsDir: "./plugins"})
	if err != nil {
		logger.Error(err, "Failed to initialize application", nil)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		if err := application.Run(); err != nil && err != http.ErrServerClosed {
			logger.Error(err, "Failed to start server", nil)
			serverErr <- err
		}
	}()

	// Wait for either interrupt signal or server error
	select {
	case <-ctx.Done():
		logger.Info("Shutting down server...", nil)
	case err := <-serverErr:
		logger.Error(err, "Server error occurred, initiating shutdown", nil)
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := application.Shutdown(shutdownCtx); err != nil {
		logger.Error(err, "Server forced to shutdown", nil)
		os.Exit(1)
	}

	logger.Info("Server exited gracefully", nil)
}
